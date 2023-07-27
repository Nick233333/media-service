package service

import (
	"bufio"
	"context"
	"crypto/md5"
	"curriculum/service/media/consumer/internal/config"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpc"
)

const (
	chanCount   = 10
	bufferCount = 1024
	charset     = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type Service struct {
	c config.Config

	waiter   sync.WaitGroup
	msgsChan chan *KafkaData
}

type KafkaData struct {
	MediaUrl  string `json:"media_url"`
	Standard  string `json:"standard"`
	NotifyUrl string `json:"notify_url"`
}

type RequestData struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func NewService(c config.Config) *Service {

	s := &Service{
		c:        c,
		msgsChan: make(chan *KafkaData),
	}
	for i := 0; i < chanCount; i++ {
		s.waiter.Add(1)
		go s.consume()
	}

	return s
}

func (s *Service) consume() {

	defer s.waiter.Done()
	for m := range s.msgsChan {

		encryptionFilePath := "./asckey/enc.keyinfo"

		// 设置视频源文件路径
		inputFile := m.MediaUrl
		timeDir := time.Now().Format("20060102") + "/"
		// 设置转码后文件路径
		outputDir := "./resource/converter/" + timeDir

		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			logx.Error("创建目录失败:", err)
			sendData(500, m.MediaUrl, "", "创建目录失败", m.NotifyUrl)
			return
		}

		parsedURL, err := url.Parse(inputFile)
		if err != nil {
			logx.Error("解析视频地址失败")
			sendData(500, m.MediaUrl, "", "解析视频地址失败", m.NotifyUrl)
			return
		}

		if parsedURL.Host != "hellocode.name" {
			downloadFileName := "./resource/" + path.Base(parsedURL.Path)

			err := downloadFile(m.MediaUrl, downloadFileName)
			if err != nil {
				logx.Error("文件下载失败")
				sendData(500, m.MediaUrl, "", "文件下载失败", m.NotifyUrl)
				return
			}
			inputFile = downloadFileName
		} else {
			inputFile = "./resource/uploads/" + path.Base(parsedURL.Path)
		}

		domainPath := "https://www.hellocode.name/converter/" + timeDir
		// 生成随机字符串
		randomString := generateRandomString(10) + getCurrentTimestamp()
		md5Hash := calculateMD5(randomString)

		m3u8FileName := md5Hash

		width, height, err := getVideoResolution(inputFile)
		if err != nil {
			logx.Error("获取视频宽度和高度失败:")
			sendData(500, m.MediaUrl, "", "获取视频宽度和高度失败", m.NotifyUrl)
			return
		}

		standard := calculateCompressedResolution(m.Standard, width, height)
		logx.Info(standard)

		compressCommand := fmt.Sprintf("ffmpeg -y -i %s -c:v libx264 -c:a copy -f hls -force_key_frames expr:gte(t,n_forced*5) -hls_time 5 %s -hls_list_size 0 -hls_key_info_file %s -hls_playlist_type vod -hls_segment_filename %s%s_%%03d.ts %s%s.m3u8", inputFile, standard, encryptionFilePath, outputDir, md5Hash, outputDir, m3u8FileName)

		if err := executeCommand(compressCommand); err != nil {
			logx.Error("压缩转码失败:", err)
			sendData(500, m.MediaUrl, "", "压缩转码失败", m.NotifyUrl)
			return
		}

		md5FileName := m3u8FileName + ".m3u8"
		md5TSFiles(outputDir, md5FileName, domainPath, m3u8FileName)

		m3u8Duration, err := getM3U8Duration(outputDir + md5FileName)
		if err != nil {
			fmt.Println("Error getting M3U8 duration:", err)
			return
		}

		mp4Duration, err := getMP4Duration(inputFile)
		if err != nil {
			fmt.Println("Error getting MP4 duration:", err)
			return
		}

		if (mp4Duration - m3u8Duration) != 0 {

			if err := executeCommand(compressCommand); err != nil {
				logx.Error("重试压缩转码失败:", err)
				sendData(500, m.MediaUrl, "", "重试压缩转码失败", m.NotifyUrl)
				return
			}
			newM3u8Duration, err := getM3U8Duration(outputDir + md5FileName)
			if err != nil {
				fmt.Println("Error getting M3U8 duration:", err)
				return
			}
			if (mp4Duration - newM3u8Duration) != 0 {
				sendData(500, m.MediaUrl, "", "视频转码时长不正确", m.NotifyUrl)
				return
			}
			md5TSFiles(outputDir, md5FileName, domainPath, m3u8FileName)
		}

		sendData(200, m.MediaUrl, domainPath+m3u8FileName+".m3u8", "success", m.NotifyUrl)

	}
}

func downloadFile(url, filePath string) error {

	resp, err := httpc.Do(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求失败，状态码：%d", resp.StatusCode)
	}

	// 创建文件
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// 将响应体内容拷贝到文件中
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func sendData(code int, mp4Url string, m3u8Url string, msg string, notifyUrl string) {
	data := &RequestData{
		Code: code,
		Msg:  msg,
		Data: map[string]interface{}{
			"media_url": mp4Url,
			"m3u8_url":  m3u8Url,
		},
	}

	resp, err := httpc.Do(context.Background(), http.MethodPost, notifyUrl, data)

	if err != nil {
		logx.Error(err)
		return
	}

	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logx.Error("unexpected response status code: ", resp.StatusCode)
	}

	logx.Info(resp.StatusCode)
	logx.Info(data)
}
func executeCommand(command string) error {
	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err
}

func (s *Service) Consume(_ string, value string) error {
	// logx.Infof("Consume value: %s\n", value)
	var kafkaData KafkaData
	if err := json.Unmarshal([]byte(value), &kafkaData); err != nil {
		return err
	}

	s.msgsChan <- &kafkaData
	return nil
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func calculateMD5(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

func getCurrentTimestamp() string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%d", timestamp)
}

func md5TSFiles(dirPath string, fieleName string, domainPath string, m3u8FileName string) {

	content, err := ioutil.ReadFile(dirPath + fieleName)
	if err != nil {
		logx.Error("无法读取文件内容")
		return
	}

	// 将文件内容转换为字符串
	contentStr := string(content)

	// 按行分割字符串
	lines := strings.Split(contentStr, "\n")
	// 处理每一行
	for i, line := range lines {
		if strings.HasSuffix(line, ".ts") {
			tsFileName := strings.TrimSpace(line)
			encryptedName := calculateMD5(tsFileName)
			updatedLine := domainPath + m3u8FileName + "_" + encryptedName + ".ts"
			lines[i] = updatedLine

			oldTsFileName := dirPath + tsFileName
			newTsFileName := dirPath + m3u8FileName + "_" + encryptedName + ".ts"
			err := os.Rename(oldTsFileName, newTsFileName)
			if err != nil {
				logx.Error("无法重命名文件：%s\n", err.Error())
				return
			}
		}
	}

	// 重新组合修改后的内容
	modifiedContent := strings.Join(lines, "\n")

	// 将修改后的内容写入文件
	err = ioutil.WriteFile(dirPath+fieleName, []byte(modifiedContent), 0644)
	if err != nil {
		logx.Error("无法写入文件")
		return
	}

}

func calculateCompressedResolution(resolution string, width, height int) string {
	switch resolution {
	case "480p":
		if width > height {
			targetWidth := 854
			targetHeight := int(float64(targetWidth) * float64(height) / float64(width))
			targetHeight = makeDivisibleByTwo(targetHeight) // 调整高度为2的倍数
			return fmt.Sprintf("%s %dx%d", "-s", targetWidth, targetHeight)
		} else {
			targetHeight := 854
			targetWidth := int(float64(targetHeight) * float64(width) / float64(height))
			targetWidth = makeDivisibleByTwo(targetWidth) // 调整宽度为2的倍数
			return fmt.Sprintf("%s %dx%d", "-s", targetWidth, targetHeight)
		}
	case "720p":
		if width > height {
			targetHeight := 720
			targetWidth := int(float64(targetHeight) * float64(width) / float64(height))
			targetWidth = makeDivisibleByTwo(targetWidth) // 调整宽度为2的倍数
			return fmt.Sprintf("%s %dx%d", "-s", targetWidth, targetHeight)
		} else {
			targetWidth := 720
			targetHeight := int(float64(targetWidth) * float64(height) / float64(width))
			targetHeight = makeDivisibleByTwo(targetHeight) // 调整高度为2的倍数
			return fmt.Sprintf("%s %dx%d", "-s", targetWidth, targetHeight)
		}
	case "1080p":
		if width > height {
			targetHeight := 1080
			targetWidth := int(float64(targetHeight) * float64(width) / float64(height))
			targetWidth = makeDivisibleByTwo(targetWidth) // 调整宽度为2的倍数
			return fmt.Sprintf("%s %dx%d", "-s", targetWidth, targetHeight)
		} else {
			targetWidth := 1080
			targetHeight := int(float64(targetWidth) * float64(height) / float64(width))
			targetHeight = makeDivisibleByTwo(targetHeight) // 调整高度为2的倍数
			return fmt.Sprintf("%s %dx%d", "-s", targetWidth, targetHeight)
		}
	default:
		return ""
	}
}

func getVideoResolution(videoPath string) (int, int, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", videoPath)
	output, err := cmd.Output()
	if err != nil {
		logx.Error(111)
		return 0, 0, fmt.Errorf("failed to execute ffprobe: %v", err)
	}
	logx.Info("aaaaaa")
	resolution := strings.Split(strings.TrimSpace(string(output)), "x")
	if len(resolution) != 2 {
		logx.Error(222)
		return 0, 0, fmt.Errorf("failed to parse video resolution")
	}

	width, err := strconv.Atoi(resolution[0])
	if err != nil {
		logx.Error(333)
		return 0, 0, fmt.Errorf("failed to parse video width: %v", err)
	}

	height, err := strconv.Atoi(resolution[1])
	if err != nil {
		logx.Error(444)
		return 0, 0, fmt.Errorf("failed to parse video height: %v", err)
	}

	return width, height, nil
}

func makeDivisibleByTwo(value int) int {
	if value%2 != 0 {
		return value - 1
	}
	return value
}

func getM3U8Duration(m3u8Path string) (float64, error) {
	file, err := os.Open(m3u8Path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var totalDuration float64
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#EXTINF:") {
			durationStr := strings.TrimPrefix(line, "#EXTINF:")
			durationStr = strings.TrimSuffix(durationStr, ",")
			duration, err := strconv.ParseFloat(durationStr, 64)
			if err != nil {
				return 0, err
			}
			totalDuration += duration
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return totalDuration, nil
}

func getMP4Duration(mp4Path string) (float64, error) {
	fmt.Println(mp4Path)
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", mp4Path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	durationString := strings.TrimSpace(string(output))
	var duration float64
	_, err = fmt.Sscanf(durationString, "%f", &duration)
	if err != nil {
		return 0, err
	}

	return duration, nil
}
