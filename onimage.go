package main

import (
	"fmt"

	"github.com/estesp/onimage/pkg/services"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	// TODO: Make logging level configurable
	logrus.SetLevel(logrus.InfoLevel)

	// channel for errors passed to each service; errors written
	// to this channel will be reported to the monitor service, if enabled
	errChan := make(chan error)

	// Read in config; looks for current working directory "onimage.toml" or
	// looks for "/etc/onimage/onimage.toml"
	viper.SetConfigName("onimage")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/onimage")
	err := viper.ReadInConfig()
	if err != nil {
		logrus.Fatalf("can't read config file: %v", err)
	}
	config := viper.AllSettings()

	// create monitor service
	monitorService, err := services.NewMonitorService(config, errChan)
	if err != nil {
		logrus.Fatalf("unable to initialize monitoring service: %v", err)
	}
	// start ping service to send heartbeats to cronitor
	go monitorService.StartPing()
	logrus.Info(" > monitor service started successfully")

	// create weather service
	weatherService, err := services.NewWeatherDataService(config, errChan)
	if err != nil {
		logrus.Fatalf("unable to initialize weather data service: %v", err)
	}
	logrus.Info(" > weather service started successfully")

	// create "today" service which handles storing sunrise/sunset and current date
	// as well as updating the S3 bucket's "index.html" with today's data
	todayService, err := services.NewTodayService(weatherService, config, errChan)
	if err != nil {
		logrus.Fatalf("unable to initialize 'today' service: %v", err)
	}

	if err = todayService.SetTodayPage(); err != nil {
		logrus.Fatalf("unable to set the initial today page view in s3: %v", err)
	}
	dateNotifier := todayService.WatchDate()
	logrus.Info(" > today page service started successfully")

	// start the web endpoint service which is called from cron entry
	// scripts that take the photos; used to determine whether to take
	// photos (between first light/last light)
	webEndpointService := services.NewWebEndpoint(todayService)

	webEndpointService.StartWebHandler()
	logrus.Info(" > endpoint for photo time capture service started successfully")

	// all dependent services are started; now start image processing

	// create the image processor service which will handle the bulk of
	// processing of each captured webcam image
	imageProcessor, err := services.NewImageProcessingService(config, errChan, todayService, weatherService)
	if err != nil {
		logrus.Fatalf("unable to initialize image processing service: %v", err)
	}
	// the today service notifier channel will be watched to update the
	//
	imageProcessor.DateChangeNotifier(dateNotifier)

	imageProcessor.StartImageHandler()

	logrus.Infof("OnImage() Processing started successfully; watching: %s\n", todayService.GetDate())

	// this will wait forever, listening for errors
	errorHandler(errChan, monitorService)
}

func errorHandler(errors chan error, monitor services.Monitor) {
	for {
		err := <-errors
		monitor.SendFailure(fmt.Sprintf("%v", err))
	}
}
