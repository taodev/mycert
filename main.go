package main

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed static/*.html
var templateFiles embed.FS

type MakeCert struct {
	Domains []string `json:"domains"`
}

var (
	// 监听地址
	address = ":8080"
	// 管理密钥
	tokenKey = "mycert"
	// 临时目录
	tempDir = "./temp"
	// mkcert 路径
	mkcertBin = "./mkcert"
	// 根证书目录
	carootDir = "./ca"
	// 网站名称
	titleName = "HTTPS 自签证书"
)

func getEnvOrDefault(envKey, fallback string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}

	return fallback
}

// 初始化命令行
func initFlags() {
	// 从环境变量中获取
	address = getEnvOrDefault("ADDR", address)
	tokenKey = getEnvOrDefault("TOKEN", tokenKey)
	tempDir = getEnvOrDefault("TEMP_DIR", tempDir)
	mkcertBin = getEnvOrDefault("MKCERT", mkcertBin)
	carootDir = getEnvOrDefault("CAROOT", carootDir)
	titleName = getEnvOrDefault("TITLE", titleName)

	// 从命令行中获取
	flag.StringVar(&address, "addr", address, "server address")
	flag.StringVar(&tokenKey, "token", tokenKey, "token key")
	flag.StringVar(&tempDir, "temp-dir", tempDir, "temp dir")
	flag.StringVar(&mkcertBin, "mkcert", mkcertBin, "mkcert path")
	flag.StringVar(&carootDir, "caroot", carootDir, "ca dir")
	flag.StringVar(&titleName, "title", titleName, "site name")
	flag.Parse()
}

// 获取完整路径
func initPath() (err error) {
	// 获取绝对路径
	if tempDir, err = filepath.Abs(tempDir); err != nil {
		return
	}

	if mkcertBin, err = filepath.Abs(mkcertBin); err != nil {
		return
	}

	if carootDir, err = filepath.Abs(carootDir); err != nil {
		return
	}

	if err = workDir(tempDir); err != nil {
		return
	}

	if err = workDir(carootDir); err != nil {
		return
	}

	return
}

func main() {
	initFlags()

	var err error
	if err = initPath(); err != nil {
		log.Fatal("initPath:", err)
	}

	if err = initRootCA(); err != nil {
		log.Fatal("initRootCA:", err)
	}

	// 初始化 gin
	router := gin.Default()

	// 开启 gzip

	// 从嵌入的文件系统中加载模板
	subFS, err := fs.Sub(templateFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	// 解析模板文件
	templ := template.Must(template.New("").ParseFS(subFS, "*.html"))

	// 设置模板到 Gin 引擎
	router.SetHTMLTemplate(templ)

	// 提取 static 文件
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	router.StaticFS("/static", http.FS(staticFS))
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Title": titleName,
		})
	})

	// /mycertCA.pem 路由到 /ca/rootCA.pem
	router.GET("/mycertCA-key.pem", func(c *gin.Context) {
		// 验证密钥
		if c.Query("token") != tokenKey {
			c.String(http.StatusUnauthorized, "Unauthorized")
			return
		}

		c.FileAttachment(filepath.Join(carootDir, "rootCA-key.pem"), "mycertCA-key.pem")
	})

	// /mycertCA.pem 路由到 /ca/rootCA.pem
	router.GET("/mycertCA.pem", func(c *gin.Context) {
		c.FileAttachment(filepath.Join(carootDir, "rootCA.pem"), "mycertCA.pem")
	})

	// /mycertCA.crt 路由到 /ca/rootCA.pem
	router.GET("/mycertCA.crt", func(c *gin.Context) {
		c.FileAttachment(filepath.Join(carootDir, "rootCA.pem"), "mycertCA.crt")
	})

	router.POST("/api/make", handleMakeCert)

	if err := router.Run(address); err != nil {
		log.Fatal("router.Run:", err)
	}
}

func handleMakeCert(c *gin.Context) {
	var req MakeCert
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Println(req.Domains)

	randomPrefix := randomName()
	certFile := randomPrefix + ".pem"
	keyFile := randomPrefix + "-key.pem"

	args := append([]string{"--cert-file", certFile, "--key-file", keyFile}, req.Domains...)

	cmd := exec.Command(mkcertBin, args...)
	cmd.Dir = tempDir
	cmd.Env = []string{"CAROOT=" + carootDir}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": out.String(),
		})
		return
	}

	certPath := filepath.Join(cmd.Dir, certFile)
	keyPath := filepath.Join(cmd.Dir, keyFile)

	cert, err := os.ReadFile(certPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"cert": string(cert),
		"key":  string(key),
	})
}

func randomName() string {
	v := time.Now().Format("20060102150405-")
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return v + hex.EncodeToString(b)
}

// 判断目录是否存在，如果不存在则创建
func workDir(path string) (err error) {
	_, err = os.Stat(path)
	if os.IsExist(err) {
		return nil
	}

	// 不存在，则创建
	err = os.MkdirAll(path, os.ModePerm)
	return
}

// 判断根证书是否存在
func rootCAExists() bool {
	_, err := os.Stat(filepath.Join(carootDir, "rootCA.pem"))
	return err == nil
}

// 初始化根证书
func initRootCA() error {
	if rootCAExists() {
		return nil
	}

	cmd := exec.Command(mkcertBin, "-install")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "CAROOT="+carootDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
