package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
	_ "time/tzdata"
)

const (
	UserAgent = "Mozilla/5.0 (compatible; monthly-playlist-generator; +https://github.com/eniehack/monthplgen)"
)

type MbidMapping struct {
	RecordingMbid string `json:"recording_mbid"`
}

type TrackMetadata struct {
	MbidMapping MbidMapping `json:"mbid_mapping"`
}

type Listen struct {
	ListenedAt    int64         `json:"listened_at"`
	TrackMetadata TrackMetadata `json:"track_metadata"`
}

type Payload struct {
	Listens []Listen `json:"listens"`
	Count   int      `json:"count"`
}

type ApiResponse struct {
	Payload Payload `json:"payload"`
}

type JspfTrack struct {
	Identifier string `json:"identifier"`
}

type JspfPlaylist struct {
	Title string      `json:"title"`
	Track []JspfTrack `json:"track"`
}

type JspfRoot struct {
	Playlist JspfPlaylist `json:"playlist"`
}

func ParseMonth(yearMonthStr string) (year int, month time.Month, err error) {
	targetMonth, err := time.Parse("2006-01", yearMonthStr)
	if err != nil {
		return 0, 0, err
	}
	return targetMonth.Year(), targetMonth.Month(), nil
}

func GuessLastDayOfMonth(year int, month time.Month) int {
	nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	lastDay := nextMonth.AddDate(0, 0, -1)
	return lastDay.Day()
}

func main() {
	var (
		flgMonth = flag.String("month", time.Now().Format("2006-01"), "target month")
		flgUser  = flag.String("user", "", "listenbrainz username")
		//flgToken    = flag.String("token", "", "listenbrainz user token")
		flgTz    = flag.String("timezone", "utc", "timezone")
		envMonth = os.Getenv("PLGEN_TARGET_MONTH")
		envUser  = os.Getenv("PLGEN_LB_USERNAME")
		//envToken    = os.Getenv("PLGEN_LB_TOKEN")
		envTimezone = os.Getenv("PLGEN_TIMEZONE")
		month       string
		user        string
		//token       string
		timezone string
	)
	flag.Parse()

	if envMonth == "" {
		month = *flgMonth
	}
	if envUser == "" {
		user = *flgUser
	}
	/*
		if envToken == "" {
			token = *flgToken
		}
	*/
	if envTimezone == "" {
		timezone = *flgTz
	}
	if user == "" {
		os.Exit(1)
	}
	/*
		if token == "" {
			os.Exit(1)
		}
	*/

	targetYear, targetMonth, err := ParseMonth(month)
	if err != nil {
		os.Exit(1)
	}
	tz, err := time.LoadLocation(timezone)
	if err != nil {
		log.Fatalf("cannot parse timezone: %s\n", err)
		os.Exit(1)
	}
	startOfMonth := time.Date(targetYear, targetMonth, 1, 0, 0, 0, 0, tz)
	endOfMonth := time.Date(targetYear, targetMonth, GuessLastDayOfMonth(targetYear, targetMonth), 0, 0, 0, 0, tz)

	url := fmt.Sprintf("https://api.listenbrainz.org/1/user/%s/listens", user)

	client := &http.Client{}
	trackCounter := make(map[string]int)
	maxTs := endOfMonth.Unix()
	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalf("Failed to create request: %v\n", err)
		}
		q := req.URL.Query()
		q.Add("max_ts", strconv.FormatInt(maxTs, 10))
		req.URL.RawQuery = q.Encode()
		req.Header.Set("User-Agent", UserAgent)
		log.Printf("%s\n", req.URL.String())

		time.Sleep(time.Second * 1)
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Failed to get response: %v\n", err)
		}
		log.Println("X-RateLimit-Limit", resp.Header.Get("X-RateLimit-Limit"))
		log.Println("X-RateLimit-Remaining", resp.Header.Get("X-RateLimit-Remaining"))
		log.Println("X-RateLimit-Reset-In", resp.Header.Get("X-RateLimit-Reset-In"))
		log.Println("X-RateLimit-Reset", resp.Header.Get("X-RateLimit-Reset"))
		defer resp.Body.Close()

		var apiResponse ApiResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			log.Fatalf("Failed to decode response: %v\n", err)
		}
		log.Printf("%d-%d: %d(%d)\n", startOfMonth.Unix(), maxTs, resp.StatusCode, apiResponse.Payload.Count)

		for _, listen := range apiResponse.Payload.Listens {
			if 1 <= len(listen.TrackMetadata.MbidMapping.RecordingMbid) {
				trackCounter[listen.TrackMetadata.MbidMapping.RecordingMbid]++
			}
		}
		log.Printf("%d < %d\n", apiResponse.Payload.Listens[apiResponse.Payload.Count-1].ListenedAt, startOfMonth.Unix())
		if apiResponse.Payload.Listens[apiResponse.Payload.Count-1].ListenedAt <= startOfMonth.Unix() {
			break
		}
		maxTs = apiResponse.Payload.Listens[apiResponse.Payload.Count-1].ListenedAt
	}

	type trackCount struct {
		Track string
		Count int
	}
	var sortedTracks []trackCount
	for track, count := range trackCounter {
		sortedTracks = append(sortedTracks, trackCount{track, count})
	}
	sort.Slice(sortedTracks, func(i, j int) bool {
		return sortedTracks[i].Count > sortedTracks[j].Count
	})

	playlist := new(JspfRoot)
	playlist.Playlist.Title = month
	for _, track := range sortedTracks {
		t := new(JspfTrack)
		t.Identifier = fmt.Sprintf("https://musicbrainz.org/recording/%s", track.Track)
		playlist.Playlist.Track = append(playlist.Playlist.Track, *t)
	}

	if err := json.NewEncoder(os.Stdout).Encode(playlist); err != nil {
		log.Fatalf("%s\n", err)
	}
}
