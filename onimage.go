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
	monitorService, err := services.NewMonitorService(config)
	if err != nil {
		logrus.Fatalf("unable to initialize monitoring service: %v", err)
	}
	monitorService.StartCronitorPing()

	weatherService, err := services.NewWeatherDataService(config)
	if err != nil {
		logrus.Fatalf("unable to initialize weather data service: %v", err)
	}

	todayService, err := services.NewTodayService(weatherService, config)
	if err != nil {
		logrus.Fatalf("unable to initialize 'today' service: %v", err)
	}

	todayService.SetTodayPage()
	dateNotifier, errChan := todayService.WatchDate()

	go errorHandler(errChan, monitorService)

	webEndpointService := services.NewWebEndpoint(todayService)

	webEndpointService.StartWebHandler()

	// all dependent services are started; now start image processing

	imageProcessor, err := services.NewImageProcessingService(config, todayService, weatherService)
	if err != nil {
		logrus.Fatalf("unable to initialize image processing service: %v", err)
	}
	imageProcessor.DateChangeNotifier(dateNotifier)

	imageProcessor.StartImageHandler()

	logrus.Infof("OnImage() Processing started successfully; watching: %s\n", todayService.GetDate())

	done := make(chan bool, 1)
	<-done
}

func errorHandler(errors chan error, monitor *services.Monitor) {
	for {
		err := <-errors
		monitor.SendFailure(fmt.Sprintf("%v", err))
	}
}
