package services

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type suntimez struct {
	Photo      int    `json:"photo"`
	Sunrise    int64  `json:"sunrise"`
	Sunset     int64  `json:"sunset"`
	NowUnix    int64  `json:"now_unix"`
	SunriseStr string `json:"sunrise_str"`
	SunsetStr  string `json:"sunset_str"`
}

type WebEndpoint struct {
	todayService *Today
}

func NewWebEndpoint(tService *Today) *WebEndpoint {
	return &WebEndpoint{
		todayService: tService,
	}
}

func (we *WebEndpoint) StartWebHandler() {
	mux := http.NewServeMux()
	mux.HandleFunc("/phototimez", we.handler)
	go we.listenerRoutine(mux)
}

func (we *WebEndpoint) listenerRoutine(mux *http.ServeMux) {
	if err := http.ListenAndServe(":5000", mux); err != nil {
		logrus.Errorf("endpoint listener failed: %v", err)
	}
}

func (we *WebEndpoint) handler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	sunrisePre := we.todayService.GetSunrise() - 1800
	sunsetPost := we.todayService.GetSunset() + 1800
	resp := 0
	if now.Unix() >= sunrisePre && now.Unix() <= sunsetPost {
		resp = 1
	}
	// since sometimes the light lingers longer than 30 min after sunset
	// use the color profile of the last photo to extend photo hours as
	// necessary
	if now.Unix() > sunsetPost && we.todayService.GetDarkPercent() < 95.0 {
		logrus.Infof("Extending photo hours; still some light (%f) at %v", we.todayService.GetDarkPercent(), now)
		resp = 1
	}
	riseTime := time.Unix(we.todayService.GetSunrise(), 0)
	setTime := time.Unix(we.todayService.GetSunset(), 0)
	sresp := suntimez{
		Photo:      resp,
		Sunrise:    we.todayService.GetSunrise(),
		Sunset:     we.todayService.GetSunset(),
		NowUnix:    now.Unix(),
		SunriseStr: riseTime.String(),
		SunsetStr:  setTime.String(),
	}
	b, err := json.Marshal(sresp)
	if err != nil {
		logrus.Errorf("can't marshal suntimes JSON: %v", err)
	}
	w.Write(b)
}
