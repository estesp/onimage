package services

import (
	"bufio"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/estesp/onimage/pkg/util"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Today struct {
	dateStr             string
	homeDir             string
	sunrise             int64
	sunset              int64
	weatherService      *WeatherData
	darkPercent         float32
	s3bucket            string
	pageTemplate        *template.Template
	pageTemplateName    string
	offlinePageTemplate string
	offline             bool
}

type PageData struct {
	Today   string
	Sunrise string
	Sunset  string
}

var (
	awscpIndexCmd = []string{"aws", "s3", "cp", "SOMEFILE", "BUCKETLOCATION", "--acl", "public-read",
		"--content-type", "text/html", "--metadata-directive", "REPLACE", "--expires"}
)

func NewTodayService(wdService *WeatherData, config map[string]interface{}) (*Today, error) {

	dateStr := util.GetDateString()

	pageTmpl, err := util.GetStringFromConfig(config, "website.page_template")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve entry 'website.page_template' from config: %w", err)
	}
	offlinePageTmpl, err := util.GetStringFromConfig(config, "website.offline_page")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve entry 'website.offline_page' from config: %w", err)
	}
	s3bucketName, err := util.GetStringFromConfig(config, "website.bucket")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve entry 'website.bucket' from config: %w", err)
	}
	homeDir, err := util.GetStringFromConfig(config, "home_dir")
	if err != nil {
		return nil, fmt.Errorf("can't retrieve entry 'home_dir' from config: %w", err)
	}
	today := &Today{
		homeDir:             homeDir,
		dateStr:             dateStr,
		weatherService:      wdService,
		darkPercent:         100.0,
		s3bucket:            s3bucketName,
		pageTemplateName:    filepath.Base(pageTmpl),
		pageTemplate:        template.Must(template.ParseFiles(pageTmpl)),
		offlinePageTemplate: offlinePageTmpl,
	}

	awscpIndexCmd[4] = fmt.Sprintf("s3://%s/index.html", s3bucketName)

	weather, err := wdService.GetCurrentWeather()
	if err != nil {
		return nil, err
	}
	today.sunrise = weather.Sys.Sunrise
	today.sunset = weather.Sys.Sunset

	return today, nil
}

func (t *Today) GetDate() string {
	return t.dateStr
}

func (t *Today) GetSunrise() int64 {
	return t.sunrise
}

func (t *Today) GetSunset() int64 {
	return t.sunset
}

func (t *Today) GetDarkPercent() float32 {
	return t.darkPercent
}

func (t *Today) SetDarkPercent(percent float32) {
	t.darkPercent = percent
}

// SetTodayPage sets up an index.html for the static site with today's date and sunrise/sunset info
func (t *Today) SetTodayPage() error {
	if t.offline {
		// if we're in "webcam offline" mode, don't set up the new page
		return nil
	}
	// change the website page with today's info
	// set up an expiration time for our index page one day from now
	// TODO: set this up to expire at midnight, not just adding 24hr to "now"
	tomorrow := time.Now().Add(24 * time.Hour)
	expires := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 6, 0, 0, 0, time.UTC).Format(http.TimeFormat)

	riseTime := time.Unix(t.GetSunrise(), 0)
	setTime := time.Unix(t.GetSunset(), 0)
	sunriseStr := fmt.Sprintf("%02d:%02d", riseTime.Hour(), riseTime.Minute())
	sunsetStr := fmt.Sprintf("%02d:%02d", setTime.Hour(), setTime.Minute())

	data := PageData{
		Today:   t.GetDate(),
		Sunrise: sunriseStr,
		Sunset:  sunsetStr,
	}
	tmpFile, err := os.CreateTemp("/tmp", "index")
	if err != nil {
		return errors.Wrap(err, "unable to create temp file for index page generation")
	}
	writer := bufio.NewWriter(tmpFile)
	if err := t.pageTemplate.ExecuteTemplate(writer, t.pageTemplateName, data); err != nil {
		return errors.Wrap(err, "unable to execute template for index page")
	}
	if err := writer.Flush(); err != nil {
		return errors.Wrap(err, "unable to flush bytes to temp file")
	}
	if err := tmpFile.Close(); err != nil {
		return errors.Wrap(err, "unable to close temp file")
	}

	awscpIndexCmd[3] = tmpFile.Name()
	out, err := util.RunCommand(t.homeDir, append(awscpIndexCmd, expires))
	if err != nil {
		logrus.Errorf("Error calling 'aws cp' from tmp file %s to S3: %v", tmpFile.Name(), err)
		logrus.Errorf(">      Command: %s", strings.Join(awscpIndexCmd, " "))
		logrus.Errorf(">  Full output: %s", out)
	}
	os.Remove(tmpFile.Name())
	return err
}

func (t *Today) SetOffline() {
	t.offline = true
}

func (t *Today) SetOnline() error {
	t.offline = false
	return t.SetTodayPage()
}

// WatchDate starts a goroutine that triggers the daily update of the index page
func (t *Today) WatchDate() (chan string, chan error) {
	dayNotifier := make(chan string)
	errNotifier := make(chan error)

	go t.watchDate(dayNotifier, errNotifier)
	return dayNotifier, errNotifier
}

func (t *Today) watchDate(notifier chan string, errors chan error) {
	tick := time.NewTicker(15 * time.Minute)
	for {
		select {
		case <-tick.C:
			today := util.GetDateString()
			if today != t.GetDate() {
				t.dateStr = today
				w, err := t.weatherService.GetCurrentWeather()
				if err != nil {
					logrus.Errorf("error retrieving sunrise/sunset for new day: %v", err)
					errors <- err
				} else {
					t.sunrise = w.Sys.Sunrise
					t.sunset = w.Sys.Sunset
					if err := t.SetTodayPage(); err != nil {
						logrus.Errorf("unable to setup index.html for new day: %v", err)
						errors <- err
					}
				}
				notifier <- t.dateStr
			}
		}
	}
}

func (t *Today) GetHomeDirectory() string {
	return t.homeDir
}
