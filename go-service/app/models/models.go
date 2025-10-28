package models

// ABR Ladder - multiple quality levels for adaptive bitrate streaming
var ABRLadder = []struct {
	Name         string
	Resolution   string
	Bitrate      string
	AudioBitrate string
}{
	{"1080p", "1920x1080", "5000k", "192k"},
	{"720p", "1280x720", "3000k", "128k"},
	{"480p", "854x480", "1500k", "128k"},
	{"360p", "640x360", "800k", "96k"},
}

// Models VideoInfo, Rendition, VideoMetadata, VideoListResponse
type VideoInfo struct {
	Duration float64 `json:"duration"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Codec    string  `json:"codec"`
}

type Rendition struct {
	Name       string `json:"name"`
	Resolution string `json:"resolution"`
	Bitrate    string `json:"bitrate"`
	URL        string `json:"url,omitempty"`
	Playlist   string `json:"playlist,omitempty"`
}

type VideoMetadata struct {
	ID                string       `json:"id"`
	Filename          string       `json:"filename"`
	Title             string       `json:"title"`
	Status            string       `json:"status"`
	Duration          float64      `json:"duration,omitempty"`
	Width             int          `json:"width,omitempty"`
	Height            int          `json:"height,omitempty"`
	CreatedAt         string       `json:"created_at"`
	MasterPlaylistURL string       `json:"master_playlist_url,omitempty"`
	Renditions        []*Rendition `json:"renditions,omitempty"`
}

type VideoListResponse struct {
	Total  int              `json:"total"`
	Videos []*VideoMetadata `json:"videos"`
}
