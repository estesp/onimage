package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	defaultBaseDir = "/home/estesp/images"
	indexTmpl      = "/home/estesp/images/index.html.tmpl"
)

var (
	tmpl          = template.Must(template.ParseFiles(indexTmpl))
	awscpIndexCmd = []string{"aws", "s3", "cp", "SOMEFILE", "s3://kwcamlive/index.html", "--acl", "public-read",
		"--content-type", "text/html", "--metadata-directive", "REPLACE", "--expires"}
)

type ProcessingService struct {
	curDate    string
	sunrise    int64
	sunset     int64
	imgBaseDir string
	watcher    *fsnotify.Watcher
}

func NewProcessingService() (*ProcessingService, error) {
	service := &ProcessingService{
		curDate:    getDate(),
		imgBaseDir: defaultBaseDir,
	}
	w, err := getWeather()
	if err != nil {
		return nil, err
	}
	service.sunrise = w.Sys.Sunrise
	service.sunset = w.Sys.Sunset

	// ensure dir is created before setting a watch
	if err := os.Mkdir(service.GetImageDir(), os.FileMode(0755)); err != nil {
		if !os.IsExist(err) {
			logrus.Errorf("unable to create dir %s: %v", service.GetImageDir(), err)
		}
	}
	if err := service.SetTodayPage(); err != nil {
		logrus.Errorf("unable to set up today's index web page: %v", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(service.GetImageDir())
	if err != nil {
		return nil, err
	}
	service.watcher = watcher
	go service.WatchDate()

	go service.StartWebHandler()
	return service, nil
}

func (p *ProcessingService) SetDate(date string) {
	p.curDate = date
}

func (p *ProcessingService) GetDate() string {
	return p.curDate
}

func (p *ProcessingService) SetSunTimes(sunrise int64, sunset int64) {
	p.sunrise = sunrise
	p.sunset = sunset
}

func (p *ProcessingService) GetSunrise() int64 {
	return p.sunrise
}

func (p *ProcessingService) GetSunset() int64 {
	return p.sunset
}

func (p *ProcessingService) GetImageDir() string {
	return fmt.Sprintf("%s/%s", p.imgBaseDir, p.curDate)
}

func (p *ProcessingService) GetWatcher() *fsnotify.Watcher {
	return p.watcher
}

func (p *ProcessingService) StartImageNotifier() chan string {
	newDirs := make(chan string)
	readyDirs := make(chan string)
	go listen(p.watcher, newDirs)
	go handleNewDirs(newDirs, readyDirs)
	return readyDirs
}

type PageData struct {
	Today   string
	Sunrise string
	Sunset  string
}

func (p *ProcessingService) WatchDate() {
	t := time.NewTicker(15 * time.Minute)
	for {
		select {
		case <-t.C:
			today := getDate()
			if today != p.GetDate() {
				if err := p.watcher.Remove(p.GetImageDir()); err != nil {
					logrus.Errorf("error removing current watched dir %s: %v", p.GetDate(), err)
				}
				p.SetDate(today)
				logrus.Infof("changing current watch folder to: %s\n", p.GetDate())
				if err := os.Mkdir(p.GetImageDir(), os.FileMode(0755)); err != nil {
					if !os.IsExist(err) {
						logrus.Errorf("error creating dir %s: %v", p.GetImageDir(), err)
					}
				}
				if err := p.watcher.Add(p.GetImageDir()); err != nil {
					logrus.Errorf("error adding new watched dir %s: %v", p.GetDate(), err)
				}
				w, err := getWeather()
				if err != nil {
					logrus.Errorf("error retrieving sunrise/sunset for new day: %v", err)
				} else {
					p.SetSunTimes(w.Sys.Sunrise, w.Sys.Sunset)
					if err := p.SetTodayPage(); err != nil {
						logrus.Errorf("unable to setup index.html for new day: %v", err)
					}
				}
			}
		}
	}
}

// SetTodayPage sets up an index.html for the static site with today's date and sunrise/sunset info
func (p *ProcessingService) SetTodayPage() error {
	// set up an expiration time for our index page ~1-2am of the next day
	t := time.Now().Add(24 * time.Hour)
	expires := time.Date(t.Year(), t.Month(), t.Day(), 6, 0, 0, 0, time.UTC).Format(http.TimeFormat)

	riseTime := time.Unix(p.GetSunrise(), 0)
	setTime := time.Unix(p.GetSunset(), 0)
	sunriseStr := fmt.Sprintf("%02d:%02d", riseTime.Hour(), riseTime.Minute())
	sunsetStr := fmt.Sprintf("%02d:%02d", setTime.Hour(), setTime.Minute())

	data := PageData{
		Today:   p.GetDate(),
		Sunrise: sunriseStr,
		Sunset:  sunsetStr,
	}
	tmpFile, err := ioutil.TempFile("/tmp", "index")
	if err != nil {
		return errors.Wrap(err, "unable to create temp file for index page generation")
	}
	writer := bufio.NewWriter(tmpFile)
	if err := tmpl.ExecuteTemplate(writer, "index.html.tmpl", data); err != nil {
		return errors.Wrap(err, "unable to execute template for index page")
	}
	if err := writer.Flush(); err != nil {
		return errors.Wrap(err, "unable to flush bytes to temp file")
	}
	if err := tmpFile.Close(); err != nil {
		return errors.Wrap(err, "unable to close temp file")
	}
	awsCmdCopy := make([]string, len(awscpIndexCmd))
	copy(awsCmdCopy, awscpIndexCmd)
	awsCmdCopy[3] = tmpFile.Name()
	out, err := runCommand(defaultBaseDir, append(awsCmdCopy, expires))
	if err != nil {
		logrus.Errorf("Error calling aws cp from tmp file %s to S3: %v", tmpFile.Name(), err)
		logrus.Errorf("Full output: %s", out)
	}
	os.Remove(tmpFile.Name())
	return err
}

func getDate() string {
	return time.Now().Format("2006-01-02")
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
