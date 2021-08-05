package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dghubble/sling"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	appIDFile = ".appid"
)

var (
	appID = ""
)

type Params struct {
	Id    int    `url:"id,omitempty"`
	AppId string `url:"appid,omitempty"`
	Units string `url:"units,omitempty"`
}

type MainSection struct {
	FeelsLike float32 `json:"feels_like"`
	Humidity  float32 `json:"humidity"`
	Pressure  int     `json:"pressure"`
	Temp      float32 `json:"temp"`
	TempMax   float32 `json:"temp_max"`
	TempMin   float32 `json:"temp_min"`
}

type SysSection struct {
	Sunrise int64 `json:"sunrise"`
	Sunset  int64 `json:"sunset"`
}

type WeatherDesc struct {
	Description string `json:"description"`
	Icon        string `json:"icon"`
	ID          int    `json:"id"`
	Main        string `json:"main"`
}

type WindSection struct {
	Deg   int     `json:"deg"`
	Gust  float64 `json:"gust"`
	Speed float64 `json:"speed"`
}

type Weather struct {
	Main        MainSection   `json:"main"`
	Sys         SysSection    `json:"sys"`
	WeatherDesc []WeatherDesc `json:"weather"`
	Wind        WindSection   `json:"wind"`
}

func init() {
	// read the app ID secret from the specified config file
	b, err := ioutil.ReadFile(appIDFile)
	if err != nil {
		logrus.Fatalf("Unable to read app ID from file: %v", err)
	}
	appID = strings.TrimSuffix(string(b), "\n")
}

func getTemp() (string, error) {
	w, err := getWeather()
	if err != nil {
		return "", errors.Wrap(err, "couldn't retrieve weather")
	}
	return fmt.Sprintf("%2.1f", w.Main.Temp), nil
}

func getWeather() (*Weather, error) {
	urlBase := "https://api.openweathermap.org/data/2.5/weather"
	params := &Params{
		Id:    4752031,
		AppId: appID,
		Units: "imperial",
	}

	var toClient = &http.Client{
		Timeout: time.Second * 10,
	}

	weather := new(Weather)
	_, err := sling.New().Client(toClient).Get(urlBase).QueryStruct(params).ReceiveSuccess(weather)
	if err != nil {
		return nil, errors.Wrap(err, "error querying openweathermap")
	}
	//resp.StatusCode
	return weather, nil
}
