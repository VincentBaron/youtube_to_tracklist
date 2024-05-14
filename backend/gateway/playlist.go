package gateway

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/gin-gonic/gin"
)

type TracklistResponse struct {
	Tracks []struct {
		TrackTitle string      `json:"trackTitle"`
		Timestamp  interface{} `json:"timestamp"`
	} `json:"tracks"`
}

type SpotifyAuthResponse struct {
	AccessToken string `json:"access_token"`
}

type ExternalUrls struct {
	Spotify string `json:"spotify"`
}

type SpotifyPlaylistResponse struct {
	// other fields...
	ID           string       `json:"id"`
	ExternalUrls ExternalUrls `json:"external_urls"`
}

type SpotifySearchResponse struct {
	Tracks struct {
		Items []struct {
			URI string `json:"uri"`
		} `json:"items"`
	} `json:"tracks"`
}

func createPlaylist(c *gin.Context) {
	SpotifyClientID, ok := c.Get("SpotifyClientID")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SpotifyClientID does not exist "})
		return

	}
	SpotifyClientID = SpotifyClientID.(string)
	SpotifyClientSecret, ok := c.Get("SpotifyClientSecret")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SpotifyClientSecret does not exist "})
		return
	}
	SpotifyClientSecret = SpotifyClientSecret.(string)

	req, err := http.NewRequest("GET", "https://www.1001tracklists.com/tracklist/1yvs6s7k/fred-again..-antro-juan-cdmx-mexico-2024-04-26.html", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	req.Header.Add("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Add("accept-language", "en-US,en;q=0.9")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("cookie", "guid=21041a5fa0ed8; shortscc=4")
	req.Header.Add("dnt", "1")
	req.Header.Add("pragma", "no-cache")
	req.Header.Add("sec-ch-ua", `"Chromium";v="123", "Not:A-Brand";v="8"`)
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("sec-ch-ua-platform", `"macOS"`)
	req.Header.Add("sec-fetch-dest", "document")
	req.Header.Add("sec-fetch-mode", "navigate")
	req.Header.Add("sec-fetch-site", "none")
	req.Header.Add("sec-fetch-user", "?1")
	req.Header.Add("upgrade-insecure-requests", "1")
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var tracks []string
	doc.Find(".tlpItem").Each(func(i int, s *goquery.Selection) {
		track := s.Find(".trackValue").Text()
		track = strings.TrimSpace(track)
		tracks = append(tracks, track)
	})

	// ...

	log.Println(SpotifyClientID.(string))
	log.Println(SpotifyClientSecret.(string))

	config := &clientcredentials.Config{
		ClientID:     SpotifyClientID.(string),
		ClientSecret: SpotifyClientSecret.(string),
		TokenURL:     spotify.TokenURL,
	}
	token, err := config.Token(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Println(token)

	client2 := spotify.Authenticator{}.NewClient(token)

	// Create a new playlist
	playlist, err := client2.CreatePlaylistForUser(SpotifyClientID.(string), "New Playlist", "New playlist description", false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Search for the track URIs and add the tracks to the playlist
	for _, track := range tracks {
		results, err := client2.Search(track, spotify.SearchTypeTrack)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if len(results.Tracks.Tracks) > 0 {
			trackID := results.Tracks.Tracks[0].ID
			_, err = client2.AddTracksToPlaylist(playlist.ID, trackID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"playlistURI": playlist.URI})
}
