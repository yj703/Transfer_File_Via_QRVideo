package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"gocv.io/x/gocv"
)

var green_lower = gocv.NewScalar(0, 100, 0, 0)
var green_upper = gocv.NewScalar(100, 255, 100, 0)
var blue_lower = gocv.NewScalar(100, 0, 0, 0)
var blue_upper = gocv.NewScalar(255, 100, 100, 0)

var cacheBuf bytes.Buffer
var isContentBase64 = true

var numberOfGreens = 0
var isInGreen = false
var dataSeq = 0

var zxingCounter = 0
var javaCounter = 0

var numError = 0

type QRCodeDataFrames struct {
	IsCurrentData      bool
	Image              *gocv.Mat
	BackupImage        *gocv.Mat
	HasData            bool
	Data               string
	NumberOfDataFrames int
	Seq                int
	TotalDataLength    int
	FileName           string
}

func (q *QRCodeDataFrames) Reset() {
	if !q.IsCurrentData {
		q.Image = nil
		q.BackupImage = nil
	}
}

func (q *QRCodeDataFrames) Exit() {
	q.IsCurrentData = false
}

func (q *QRCodeDataFrames) ProcessFrame(img gocv.Mat, seq *int) {
	if !q.IsCurrentData {
		if q.Image != nil {
			q.Image.Close()
		}
		q.Image = nil
		if q.BackupImage != nil {
			q.BackupImage.Close()
		}
		q.BackupImage = nil
		q.IsCurrentData = true
		q.HasData = false
		q.Data = ""
		q.NumberOfDataFrames = 0
		*seq++
		q.Seq = *seq
		fmt.Print(">")
	} else {
		fmt.Print("@")
	}

	q.IsCurrentData = true
	if q.BackupImage != nil {
		if q.Image != nil {
			q.Image.Close()
		}
		q.Image = q.BackupImage

	}
	saveImg := img.Clone()
	q.BackupImage = &saveImg

	q.NumberOfDataFrames++
}

func (q *QRCodeDataFrames) DetectData(qrDetector *gocv.QRCodeDetector, points *gocv.Mat, qrMat *gocv.Mat) {
	data := DetectData(q.BackupImage, qrDetector, points, qrMat)
	if data != "" {
		q.HasData = true
		q.Data = data
	} else {
		data := DetectData(q.Image, qrDetector, points, qrMat)
		if data != "" {
			q.HasData = true
			q.Data = data
		} else {
			q.HasData = false
			q.Data = ""
		}
	}
	if q.HasData {
		if strings.HasPrefix(q.Data, `_%seq%_`) {
			q.Data = q.Data[len(`_%seq%_`):]
			pos := strings.Index(q.Data, "|")
			seqStr := ""
			totalSeqStr := ""
			if pos > 0 {
				seqStr = q.Data[:pos]
				q.Data = q.Data[pos+1:]
			}
			pos = strings.Index(q.Data, "|")
			if pos > 0 {
				totalSeqStr = q.Data[:pos]
				q.Data = q.Data[pos+1:]
			}
			scanSeq, _ := strconv.Atoi(seqStr)
			scanTotalSeq, _ := strconv.Atoi(totalSeqStr)
			if q.Seq != scanSeq {
				numError++
				fmt.Print("\nError: Failed to detect data - order error.  please check capture viedo quality, badx.jpg file and retry capture with slower refresh delay.\n")
			}
			if scanTotalSeq != len(q.Data) {
				numError++
				fmt.Print("\nError: Failed to detect data - length error.  please check capture viedo quality, badx.jpg file and retry capture with slower refresh delay.\n")
			}
			q.TotalDataLength += scanTotalSeq
		}

		if q.Seq == 1 {
			if strings.HasPrefix(q.Data, "_fname_") {
				q.Data = q.Data[len(`_fname_`):]
				pos := strings.Index(q.Data, "|")
				if pos > 0 {
					q.FileName = q.Data[:pos]
					q.Data = q.Data[pos+1:]
				}
			}
		}

	}
}

func DetectData(img *gocv.Mat, qrDetector *gocv.QRCodeDetector, points *gocv.Mat, qrMat *gocv.Mat) string {
	if img == nil || img.Empty() {
		fmt.Print("e")
		return ""
	}
	if qrDetector.Detect(*img, points) {
		data := qrDetector.Decode(*img, *points, qrMat)
		if data != "" {
			fmt.Print(".")
			return data
		} else {
			data = TryZxing(*img)
			if data != "" {
				fmt.Print(",")
				return data
			} else {
				fmt.Print("*")
				numError++
				fmt.Print("\nError: Failed to detect data frame.  please check capture viedo quality, badx.jpg file and retry capture with slower refresh delay.\n")
			}
		}
	} else {
		data := TryZxing(*img)
		if data != "" {
			fmt.Print(",")
			return data
		} else {
			fmt.Print("#")
			numError++
			fmt.Print("\nError: Failed to detect data frame.  please check capture viedo quality, badx.jpg file and retry capture with slower refresh delay.\n")
		}
	}
	return ""
}

func (q *QRCodeDataFrames) OutputData(qrDetector *gocv.QRCodeDetector, points *gocv.Mat, qrMat *gocv.Mat) bool {
	ret := true
	if q.IsCurrentData {
		q.Exit()
		q.DetectData(qrDetector, points, qrMat)
		if q.HasData {
			WriteToFileCache(q.Data)
			fmt.Print("-")
		} else {
			ret = false
		}
	} else {
		fmt.Print("|")
	}
	return ret
}

func threshhold_amount(image_hsv *gocv.Mat, lower, upper gocv.Scalar) int {
	sizes := image_hsv.Size()
	if len(sizes) < 2 {
		return 0
	}
	for i := range sizes {
		if sizes[i] <= 0 {
			return 0
		}
	}
	image_threshed := gocv.NewMatWithSizes(image_hsv.Size(), image_hsv.Type())
	defer image_threshed.Close()

	err := gocv.InRangeWithScalar(*image_hsv, lower, upper, &image_threshed)
	if err != nil {
		return 0
	}

	return gocv.CountNonZero(image_threshed)
}

func green_amount(image_hsv *gocv.Mat) int {
	return threshhold_amount(image_hsv, green_lower, green_upper)
}

func blue_amount(image_hsv *gocv.Mat) int {
	return threshhold_amount(image_hsv, blue_lower, blue_upper)
}

func isGreen(image_hsv *gocv.Mat) bool {
	size := image_hsv.Size()
	totalPixels := size[0] * size[1]
	if totalPixels == 0 {
		return false
	}

	return green_amount(image_hsv) > totalPixels/10
}

func isBlue(image_hsv *gocv.Mat) bool {
	size := image_hsv.Size()
	totalPixels := size[0] * size[1]

	if totalPixels == 0 {
		return false
	}
	return blue_amount(image_hsv) > totalPixels/10
}

func WriteToFileCache(data string) {
	_, err := cacheBuf.Write([]byte(data))
	if err != nil {
		log.Printf("Error writing to output cache: %v", err)
		os.Exit(1)
	}
}

func WriteToFile(filename string) error {

	if len(os.Args) > 2 && os.Args[2] != "" {
		fInfo, err := os.Stat(os.Args[2])
		if err == nil {
			if fInfo.IsDir() {
				filename = path.Join(os.Args[2], filename)
			} else {
				filename = os.Args[2]
			}
		}
		if errors.Is(err, fs.ErrNotExist) {
			filename = os.Args[2]
		}

	}

	fmt.Printf("Output file: %v\n", filename)

	outputFile, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	contentBytes := cacheBuf.Bytes()
	if isContentBase64 {
		decodedBytes := make([]byte, base64.StdEncoding.DecodedLen(len(contentBytes)))
		_, err := base64.StdEncoding.Decode(decodedBytes, contentBytes)
		if err != nil {
			log.Printf("Error decoding base64 content: %v", err)
		} else {
			contentBytes = decodedBytes
		}
	}
	_, err = outputFile.Write(contentBytes)
	if err != nil {
		log.Printf("Error writing to output file: %v", err)
		os.Exit(1)
	}
	return nil
}

func saveImage(img gocv.Mat, filename string) error {
	success := gocv.IMWrite(filename, img)
	if !success {
		return fmt.Errorf("failed to save image to %s", filename)
	}
	return nil
}

func TryZxing(img gocv.Mat) string {

	stdImg, err := img.ToImage()
	if err != nil {
		return ""
	}

	bmp, _ := gozxing.NewBinaryBitmapFromImage(stdImg)

	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		data := TryJava(img)
		if data != "" {
			javaCounter++
			fmt.Print("j")
		}
		return data
	} else {
		zxingCounter++
		return result.String()
	}
}

func TryJava(img gocv.Mat) string {

	tempPath := os.TempDir()
	tempFile := path.Join(tempPath, "temp.jpg")
	err := saveImage(img, tempFile)

	if err != nil {
		fmt.Printf("Error saving image: %v\n", err)
		return ""
	}

	out, err := exec.Command("java", "-cp", "./app-all.jar", "org.example.App", tempFile).Output()
	if err != nil {
		fmt.Printf("Error executing command: %v\n", err)
		return ""
	}
	return string(out)

}

func main() {

	if len(os.Args) < 2 {
		log.Fatalf("Usage: qrvideo [video file] [output dir/file]")
	}

	var err error
	var badFileCount int

	vFile := os.Args[1]

	if strings.Contains(strings.ToLower(vFile), "binary") {
		isContentBase64 = false
	}

	base64Env := os.Getenv("QRCODE_BASE64_CONTENT")
	if base64Env == "false" || base64Env == "0" {
		isContentBase64 = false
	}

	vCap, err := gocv.VideoCaptureFile(vFile)
	if err != nil {
		log.Fatalf("Error opening video file: %v", err)
	}
	defer vCap.Close()

	frameTotalCount := vCap.Get(gocv.VideoCaptureFrameCount)
	fmt.Printf("Total frames in video: %v\n", frameTotalCount)

	timeStart := time.Now()

	img := gocv.NewMat()
	defer img.Close()
	points := gocv.NewMat()
	qrMat := gocv.NewMat()
	defer points.Close()
	defer qrMat.Close()
	qrDetector := gocv.NewQRCodeDetector()
	defer qrDetector.Close()
	frameCount := 0

	previousPercent := 0.0

	DataProcessor := QRCodeDataFrames{}

	for vCap.IsOpened() {

		if vCap.Read(&img) {
			if !img.Empty() {
				frameCount++
				percent := float64(frameCount) / frameTotalCount * 100
				if percent-previousPercent >= 5 {
					fmt.Printf("\n--%.2f%% processed--\n ", percent)
					previousPercent = percent
				}

				//fmt.Printf("Processing frame...%v\n", frameCount)
				if isGreen(&img) {
					if !isInGreen {
						numberOfGreens++
					}
					//fmt.Printf("green frame detected. Frame:%v\n", frameCount)

					isInGreen = true

					if !DataProcessor.OutputData(&qrDetector, &points, &qrMat) {
						fmt.Printf("\nWarning: Detected green frames without QR code data. Frame: %v\n", frameCount)
						if DataProcessor.BackupImage != nil {
							saveImage(*DataProcessor.BackupImage, fmt.Sprintf("bad%v.jpg", badFileCount))
						} else {
							saveImage(img, fmt.Sprintf("bad%v.jpg", badFileCount))
						}
						badFileCount++
					}
					continue
				}
				if isBlue(&img) {

					if !DataProcessor.OutputData(&qrDetector, &points, &qrMat) {
						fmt.Printf("\nWarning: Detected blue frames without QR code data. Frame: %v\n", frameCount)
						if DataProcessor.BackupImage != nil {
							saveImage(*DataProcessor.BackupImage, fmt.Sprintf("bad%v.jpg", badFileCount))
						} else {
							saveImage(img, fmt.Sprintf("bad%v.jpg", badFileCount))
						}
						badFileCount++
					}
					fmt.Printf("\nFinished processing video. Total frames: %v\n", frameCount)
					err = WriteToFile(DataProcessor.FileName)
					if err != nil {
						numError++
						log.Printf("Error writing to file: %v", err)
					}
					break
				}

				isInGreen = false
				DataProcessor.ProcessFrame(img, &dataSeq)

			}
		} else {
			fmt.Println("End of video file,  but no blue frame detected. Total frames: ", frameCount)
			break
		}

	}

	timeEnd := time.Now()
	fmt.Printf("Total processing time: %v seconds, Total green frames detected: %v, last sequence: %v, total data length: %v\n", timeEnd.Sub(timeStart).Seconds(), numberOfGreens, dataSeq, DataProcessor.TotalDataLength)
	fmt.Printf("Detection retry rate: retry with zxing: %v - retried with java: %v\n", zxingCounter, javaCounter)
	if numError > 0 {
		fmt.Println("Error occur during video processing. ")
	} else {
		fmt.Println("Video processing completed successfully. ")
	}

}
