package main

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"flag"
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

type MakeCert struct {
	Domains []string `json:"domains"`
}

var (
	addrFlag = ":8080"
	certDir  = "./certs"
)

func main() {
	flag.StringVar(&addrFlag, "addr", ":8080", "server address")
	flag.StringVar(&certDir, "dir", certDir, "certs dir")
	flag.Parse()

	if err := workDir(certDir); err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	// 提取 static 文件
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	router.StaticFS("/static", http.FS(staticFS))
	router.GET("/", func(c *gin.Context) {
		content, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load index.html")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	})

	router.POST("/api/make", handleMakeCert)

	if err := router.Run(addrFlag); err != nil {
		log.Fatal(err)
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

	cmd := exec.Command("mkcert", args...)
	cmd.Dir = certDir

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
