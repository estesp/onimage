package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// photo every 3 minutes
	photoFreq = 3
)

var (
	replaceNNNN = regexp.MustCompile(`NNNN`)
	enfuseCmd   = []string{"enfuse", "-o", "prefinal.jpg", "01.jpg", "02.jpg", "03.jpg", "04.jpg", "05.jpg"}
	overlayCmd  = []string{"convert", "prefinal.jpg", "-pointsize", "36",
		"-draw", "gravity southwest fill white text 20,20 'NNNN' ",
		"-draw", "gravity southeast fill white text 20,20 'NNNN' ", "-pointsize", "28",
		"-draw", "gravity south fill white text 0,20 'kwcam.live' ", "final.jpg"}
	awscpCmd = []string{"aws", "s3", "cp", "final.jpg", "s3://kwcamlive/latest.jpg", "--acl", "public-read",
		"--metadata-directive", "REPLACE", "--expires"}
)

func main() {
	logrus.SetLevel(logrus.InfoLevel)

	pService, err := NewProcessingService()
	if err != nil {
		logrus.Fatalf("Couldn't create processing service: %v", err)
	}
	logrus.Infof("Current date: %s\n", pService.GetDate())

	newDirs := pService.StartImageNotifier()
	done := make(chan bool, 1)

	processImages(newDirs)
	<-done
}

func processImages(newDirs chan string) {
	for {
		dir := <-newDirs
		// create final image (enfuse)
		enfuseImages(dir)
		// overlay text: date/time, temp
		overlayImage(dir)
		// copy latest to S3 bucket for kwcam.live
		copyImagetoS3(dir)
	}
}

func enfuseImages(dir string) {
	out, err := runCommand(dir, enfuseCmd)
	if err != nil {
		logrus.Errorf("Error calling enfuse on %s: %v", dir, err)
		logrus.Errorf("Full output: %s", out)
	}
}

func overlayImage(dir string) {
	tempStr, err := getTemp()
	if err != nil {
		logrus.Errorf("can't get temp: %v", err)
		tempStr = ""
	}
	tempStr = fmt.Sprintf("%s°F", tempStr)
	timeStr := datetimeFromDir(dir)
	logrus.Infof("timestamp for image: %s", timeStr)
	logrus.Infof("current temp value: %s", tempStr)
	overlayCmdCopy := make([]string, len(overlayCmd))
	copy(overlayCmdCopy, overlayCmd)
	overlayCmdCopy[5] = replaceNNNN.ReplaceAllLiteralString(overlayCmdCopy[5], timeStr)
	overlayCmdCopy[7] = replaceNNNN.ReplaceAllLiteralString(overlayCmdCopy[7], tempStr)
	out, err := runCommand(dir, overlayCmdCopy)
	if err != nil {
		logrus.Errorf("Error calling convert on %s: %v", dir, err)
		logrus.Errorf("Full output: %s", out)
	}
}

func copyImagetoS3(dir string) {
	expiresTime := time.Now().Add(photoFreq * time.Minute).UTC()
	out, err := runCommand(dir, append(awscpCmd, expiresTime.Format(http.TimeFormat)))
	if err != nil {
		logrus.Errorf("Error calling aws cp on %s: %v", dir, err)
		logrus.Errorf("Full output: %s", out)
	}
}

func datetimeFromDir(dir string) string {
	parts := strings.Split(dir, "/")
	timestamp := parts[len(parts)-1]
	datestr := parts[len(parts)-2]

	return fmt.Sprintf("%s @ %s:%s", datestr, timestamp[:2], timestamp[2:4])
}

func runCommand(workdir string, command []string) (string, error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = append(cmd.Env, "HOME=/home/estesp")
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
