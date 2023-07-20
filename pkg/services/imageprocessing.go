package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/estesp/onimage/pkg/util"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

type ImageProcessor struct {
	imagesBaseDir  string
	s3bucket       string
	frequency      time.Duration
	todayService   *Today
	weatherService *WeatherData
	watcher        *fsnotify.Watcher
}

type ColorJson struct {
	BlackPercent float32 `json:"black_percent"`
	Colors       []struct {
		Color   []float32 `json:"color"`
		Percent float32   `json:"percent"`
	} `json:"colors"`
}

var (
	replaceNNNN = regexp.MustCompile(`NNNN`)

	enfuseCmd = []string{"enfuse", "-o", "prefinal.jpg", "01.jpg", "02.jpg", "03.jpg", "04.jpg", "05.jpg"}

	overlayCmd = []string{"convert", "prefinal.jpg", "-pointsize", "36",
		"-draw", "gravity southwest fill white text 20,20 'NNNN' ",
		"-draw", "gravity southeast fill white text 20,20 'NNNN' ", "-pointsize", "28",
		"-draw", "gravity south fill white text 0,20 'kwcam.live' ", "final.jpg"}

	awscpCmd = []string{"aws", "s3", "cp", "final.jpg", "s3://kwcamlive/latest.jpg", "--acl", "public-read",
		"--metadata-directive", "REPLACE", "--expires"}

	assessDarkCmd = []string{"sudo", "ctr", "run", "--rm", "--mount", "type=bind,src=NNNN,dst=/mnt,options=rbind:ro",
		"docker.io/estesp/opencv2:4.8.0", "ocv2", "python", "color_percents.py", "/mnt/final.jpg"}
)

func NewImageProcessingService(config map[string]interface{}, todayService *Today, weatherService *WeatherData) (*ImageProcessor, error) {
	baseDir, ok := config["images.directory"].(string)
	if !ok {
		return nil, fmt.Errorf("config file has no string entry for 'images.directory'")
	}
	freq, ok := config["images.photo_frequency"].(int)
	if !ok {
		return nil, fmt.Errorf("config file has no int entry for 'images.photo_frequency'")
	}
	s3bucketName, ok := config["website.bucket"].(string)
	if !ok {
		return nil, fmt.Errorf("config file has no string entry for 'website.bucket'")
	}
	return &ImageProcessor{
		todayService:   todayService,
		weatherService: weatherService,
		imagesBaseDir:  baseDir,
		frequency:      time.Duration(freq) * time.Minute,
		s3bucket:       s3bucketName,
	}, nil
}

func (ip *ImageProcessor) DateChangeNotifier(notifier chan string) {
	go ip.watchDate(notifier)
}

func (ip *ImageProcessor) StartImageHandler() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Fatalf("unable to create fsnotify watcher: %v", err)
	}
	if err := os.Mkdir(ip.getImageDir(), os.FileMode(0755)); err != nil {
		if !os.IsExist(err) {
			logrus.Errorf("error creating watching dir %s: %v", ip.getImageDir(), err)
		}
	}
	err = watcher.Add(ip.getImageDir())
	if err != nil {
		logrus.Fatalf("unable to add image base directory to fsnotify watcher: %v", err)
	}
	ip.watcher = watcher
	go ip.processImages()
}

func (ip *ImageProcessor) watchDate(notifier chan string) {
	for {
		newDate := <-notifier
		logrus.Infof("New day %s; changing current watch folder to: %s\n", newDate, ip.getImageDir())
		if err := os.Mkdir(ip.getImageDir(), os.FileMode(0755)); err != nil {
			if !os.IsExist(err) {
				logrus.Errorf("error creating dir %s: %v", ip.getImageDir(), err)
			}
		}
		curList := ip.watcher.WatchList()
		if len(curList) < 1 {
			logrus.Errorf("watcher should have one directory watched at all times; entries: %d", len(curList))
		} else {
			ip.watcher.Remove(curList[0])
		}
		if err := ip.watcher.Add(ip.getImageDir()); err != nil {
			logrus.Errorf("error adding new watched dir %s: %v", ip.getImageDir(), err)
		}
	}
}

func (ip *ImageProcessor) processImages() {
	newDirs := make(chan string)
	readyDirs := make(chan string)
	go listen(ip.watcher, newDirs)
	go handleNewDirs(newDirs, readyDirs)

	for {
		dir := <-readyDirs
		// create final image (enfuse)
		ip.enfuseImages(dir)
		// overlay text: date/time, temp
		ip.overlayImage(dir)
		// copy latest to S3 bucket for kwcam.live
		ip.copyImagetoS3(dir)
		// assess percent dark in image
		ip.assessDarkPercent(dir)
	}
}

func (ip *ImageProcessor) getImageDir() string {
	return fmt.Sprintf("%s/%s", ip.imagesBaseDir, ip.todayService.GetDate())
}

func (ip *ImageProcessor) enfuseImages(dir string) {
	out, err := util.RunCommand(dir, enfuseCmd)
	if err != nil {
		logrus.Errorf("Error calling enfuse on %s: %v", dir, err)
		logrus.Errorf("Full output: %s", out)
	}
}

func (ip *ImageProcessor) overlayImage(dir string) {
	tempStr, err := ip.weatherService.GetCurrentTempStr()
	if err != nil {
		logrus.Errorf("can't get temp: %v", err)
		tempStr = ""
	}
	tempStr = fmt.Sprintf("%s°F", tempStr)
	timeStr := util.DatetimeFromDir(dir)
	logrus.Infof("timestamp for image: %s", timeStr)
	logrus.Infof("current temp value: %s", tempStr)
	overlayCmdCopy := make([]string, len(overlayCmd))
	copy(overlayCmdCopy, overlayCmd)
	overlayCmdCopy[5] = replaceNNNN.ReplaceAllLiteralString(overlayCmdCopy[5], timeStr)
	overlayCmdCopy[7] = replaceNNNN.ReplaceAllLiteralString(overlayCmdCopy[7], tempStr)
	out, err := util.RunCommand(dir, overlayCmdCopy)
	if err != nil {
		logrus.Errorf("Error calling convert on %s: %v", dir, err)
		logrus.Errorf("Full output: %s", out)
	}
}

func (ip *ImageProcessor) copyImagetoS3(dir string) {
	expiresTime := time.Now().Add(ip.frequency).UTC()
	out, err := util.RunCommand(dir, append(awscpCmd, expiresTime.Format(http.TimeFormat)))
	if err != nil {
		logrus.Errorf("Error calling aws cp on %s: %v", dir, err)
		logrus.Errorf("Full output: %s", out)
	}
}

func (ip *ImageProcessor) assessDarkPercent(dir string) {
	assessCmdCopy := make([]string, len(assessDarkCmd))
	copy(assessCmdCopy, assessDarkCmd)
	assessCmdCopy[4] = replaceNNNN.ReplaceAllLiteralString(assessCmdCopy[4], dir)
	out, err := util.RunCommand(dir, assessCmdCopy)
	if err != nil {
		logrus.Errorf("Error calling opencv2 container on %s: %v", dir, err)
		logrus.Errorf("Full output: %s", out)
		return
	}
	if err = os.WriteFile(path.Join(dir, "colors.json"), []byte(out), 0644); err != nil {
		logrus.Errorf("Error writing color JSON output to file: %v", err)
	}
	var colorJson ColorJson
	if err = json.Unmarshal([]byte(out), &colorJson); err != nil {
		logrus.Errorf("Error unmarshalling JSON to Go type: %v", err)
		return
	}
	ip.todayService.SetDarkPercent(colorJson.BlackPercent)
}

func listen(w *fsnotify.Watcher, newDirs chan string) {
	for {
		e := <-w.Events
		logrus.Infof("Event: %+v\n", e)
		if e.Op == fsnotify.Create {
			fi, err := os.Stat(e.Name)
			if err != nil {
				logrus.Errorf("unable to stat %s: %v", e.Name, err)
				return
			}
			if fi.IsDir() {
				newDirs <- e.Name
			}
		}
	}
}

func handleNewDirs(newDirs chan string, readyDir chan string) {
	for {
		dir := <-newDirs
		doneFile := fmt.Sprintf("%s/done.txt", dir)
		err := waitForDone(doneFile, 15*time.Second)
		if err == nil {
			logrus.Infof("Ready for processing: %s", dir)
			readyDir <- dir
		} else {
			logrus.Errorf("timed out waiting for %s/done.txt", dir)
		}
	}
}

func waitForDone(fname string, timeout time.Duration) error {

	c := make(chan []struct{}, 1)
	go pollDoneFile(fname, c)
	select {
	case <-c:
		return nil
	case <-time.After(timeout):
		return errors.New("file not ready")
	}
}

func pollDoneFile(fname string, c chan []struct{}) {
	for {
		_, err := os.Stat(fname)
		if err == nil {
			c <- []struct{}{}
			break
		}
		time.Sleep(1 * time.Second)
	}
}