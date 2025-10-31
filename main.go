package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yabetsu93/hls-converter-api/helper"
	"github.com/yabetsu93/hls-converter-api/models"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// getEnv retrieves the value of the environment variable named by the key.
// If the variable is empty or not present, it returns the defaultValue.
// getEnv retrieves the value of the environment variable named by the key.
// If the variable is empty or not present, it returns the defaultValue.
func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// Configuration
var (
	bucketName       = getEnv("BUCKET_NAME", "assignment-bucket")
	localStoragePath = "/tmp/videos"
	port             = getEnv("PORT", "8080")
)

// GCS Client
var storageClient *storage.Client
var bucket *storage.BucketHandle

func init() {
	// Create local storage directory
	if err := os.MkdirAll(localStoragePath, 0755); err != nil {
		log.Fatalf("Failed to create a local storage directory: %v", err)
	}
}

func main() {
	// Initialize GCS client
	ctx := context.Background()
	var err error

	// Try to initialize with default credentials
	storageClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Printf("Warning: Failed to initialize GCS client with default credentials: %v", err)
		log.Printf("Will attempt to use service account from environment")

		// Try with service account if json is provided
		if saPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); saPath != "" {
			storageClient, err = storage.NewClient(ctx, option.WithCredentialsFile(saPath))
			if err != nil {
				log.Fatalf("Failed to initialize GCS client: %v", err)
			}
		}
	}

	if storageClient != nil {
		bucket = storageClient.Bucket(bucketName)
		log.Printf("Initialized GCS client with bucket: %s", bucketName)
	} else {
		log.Printf("Warning: Running without GCS client - storage operations will fail")
	}

	// Setup Gin router
	router := gin.Default()

	// Add cors middleware
	router.Use(helper.CorsMiddleware())

	// Health Check API
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "HLS Video Converter API",
			"status":  "running",
			"version": "0.0.1",
		})
	})

	// API Endpoints
	router.POST("/videos", uploadVideo)
	router.GET("/videos", listVideos)
	router.GET("/videos/:id", getVideo)

	// Start Server
	log.Printf("Starting server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server : %v", err)
	}

}

func getVideo(c *gin.Context) {
	ctx := c.Request.Context()
	videoID := c.Param("id")

	metadata, err := loadMetadataStreaming(ctx, videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}

	// Refresh signed URLs (expired after 60minutes)
	if metadata.MasterPlaylistURL != "" {
		gcsMasterPath := fmt.Sprintf("videos/%s/master.m3u8", videoID)
		if url, err := getSignedURL(ctx, gcsMasterPath, 60); err == nil {
			metadata.MasterPlaylistURL = url
		}
	}

	for _, rendition := range metadata.Renditions {
		renditionPath := fmt.Sprintf("video/%s/%s", videoID, rendition.Playlist)
		if url, err := getSignedURL(ctx, renditionPath, 60); err == nil {
			rendition.URL = url
		}
	}

	c.JSON(http.StatusOK, metadata)
}

// loadMetadataStreaming loads video metadata from GCS
func loadMetadataStreaming(ctx context.Context, videoID string) (*models.VideoMetadata, error) {
	if storageClient == nil {
		return nil, fmt.Errorf("GCS client not initialized")
	}

	blobName := fmt.Sprintf("metadata/%s.json", videoID)
	obj := bucket.Object(blobName)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %v", err)
	}
	defer reader.Close()

	// Read with size limit to prevent memory exhaustion
	const maxMetadataSize = 10 * 1024 * 1024 // 10 mb limit
	limitedReader := io.LimitReader(reader, maxMetadataSize)

	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata content: %v", err)
	}

	var metadata models.VideoMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %v", err)
	}

	return &metadata, nil
}

// GetSignedURL generates a signed URL for a GCS object
func getSignedURL(ctx context.Context, blobPath string, expirationMinutes int) (string, error) {
	if storageClient == nil {
		return "", fmt.Errorf("GCS client not initialized")
	}

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(time.Duration(expirationMinutes) * time.Minute),
	}

	url, err := bucket.SignedURL(blobPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %v", err)
	}

	return url, nil
}

func listVideos(c *gin.Context) {
	ctx := c.Request.Context()

	// query params
	query := c.Query("q")
	limit := 19
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Load all videos
	allVideos, err := listAllVideos(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to list videos: %v", err)})
		return
	}

	var filteredVideos []*models.VideoMetadata
	if query != "" {
		queryLower := strings.ToLower(query)
		for _, video := range allVideos {
			if strings.Contains(strings.ToLower(video.Filename), queryLower) || strings.Contains(strings.ToLower(video.Title), queryLower) {
				filteredVideos = append(filteredVideos, video)
			}
		}
	} else {
		filteredVideos = allVideos
	}

	// Apply offset and limit (pagination)
	total := len(filteredVideos)
	start := offset
	end := offset + limit

	if start > total {
		start = total
	}

	if end > total {
		end = total
	}

	paginatedVideos := filteredVideos[start:end]

	c.JSON(http.StatusOK, gin.H{
		"videos": paginatedVideos,
		"total":  len(filteredVideos),
	})
}

// listAllVideos lists all video metadata from GCS
func listAllVideos(ctx context.Context) ([]*models.VideoMetadata, error) {
	if storageClient == nil {
		return nil, fmt.Errorf("GCS client not initialized")
	}

	var videos []*models.VideoMetadata

	query := &storage.Query{Prefix: "metadata/"}
	it := bucket.Objects(ctx, query)

	// Process in chunks to avoid loading all at once
	const chunkSize = 100
	chunk := make([]*models.VideoMetadata, 0, chunkSize)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			// add remaining items in chunk
			if len(chunk) > 0 {
				videos = append(videos, chunk...)
			}
			break
		}
		if err != nil {
			log.Printf("Error iterating objects: %v", err)
			continue
		}

		if !strings.HasSuffix(attrs.Name, ".json") {
			continue
		}

		videoID := strings.TrimSuffix(strings.TrimPrefix(attrs.Name, "metadata/"), ".json")
		metadata, err := loadMetadataStreaming(ctx, videoID)
		if err != nil {
			log.Printf("Error loading metadata for %s: %v", videoID, err)
			continue
		}

		// add to chunk
		chunk = append(videos, metadata)

		// if chunk is full, append to results and reset
		if len(chunk) >= chunkSize {
			videos = append(videos, chunk...)
			chunk = make([]*models.VideoMetadata, 0, chunkSize)
		}
	}

	log.Print("Loaded %d videos using streaming approach", len(videos))
	return videos, nil
}

func uploadVideo(c *gin.Context) {
	ctx := c.Request.Context()

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".mp4") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only mp4 files are supported"})
		return
	}

	// Validate optional title
	title := c.PostForm("title")
	if title == "" {
		title = file.Filename
	}

	// Generate unique video ID
	videoID := uuid.New().String()

	// Save uploaded file
	inputPath := filepath.Join(localStoragePath, fmt.Sprintf("%s_input.mp4", videoID))
	if err := c.SaveUploadedFile(file, inputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file: %v", err)})
		return
	}
	defer os.Remove(inputPath)

	log.Printf("Saved uploaded file to %s", inputPath)

	// Convert to HLS
	renditions, outputDir, videoInfo, err := convertToHLS(videoID, inputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to convert video: %v", err)})
		return
	}
	defer os.RemoveAll(outputDir)

	// Upload to GCS
	gcsMastersPath, err := uploadToGCS(ctx, videoID, outputDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to upload to gcs: %v", err)})
		return
	}

	// Generate signed URLS
	masterURL, err := getSignedURL(ctx, gcsMastersPath, 60)
	if err != nil {
		log.Printf("warning failed to generate signed url: %v", err)
		masterURL = fmt.Sprintf("gs://%s/%s", bucketName, gcsMastersPath)
	}

	// Add signed URls to renditions
	for _, rendition := range renditions {
		renditionPath := fmt.Sprintf("videos/%s/%s", videoID, rendition.Playlist)
		url, err := getSignedURL(ctx, renditionPath, 60)
		if err != nil {
			log.Printf("warning: failed to generate signed url for rention: %v", err)
			url = fmt.Sprintf("gs://%s/%s", bucketName, renditionPath)
		}
		rendition.URL = url
	}

	// Create metadata
	metadata := &models.VideoMetadata{
		ID:                videoID,
		Filename:          file.Filename,
		Title:             title,
		Status:            "completed",
		Duration:          videoInfo.Duration,
		Width:             videoInfo.Width,
		Height:            videoInfo.Height,
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
		MasterPlaylistURL: masterURL,
		Renditions:        renditions,
	}

	// Save metadata
	if err := saveMetadata(ctx, videoID, metadata); err != nil {
		log.Printf("warning: failed to save metadata: %v", err)
	}

	log.Printf("successfully processed video: %s", videoID)

	c.JSON(http.StatusOK, metadata)
}

// convert to hls converts mp4 to hls with ABR ladder
func convertToHLS(videoID, inputPath string) ([]*models.Rendition, string, *models.VideoInfo, error) {
	outputDir := filepath.Join(localStoragePath, videoID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, "", nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	log.Printf("Starting HLS conversion for video %s", videoID)

	// Get video info
	videoInfo, err := getVideoInfo(inputPath)
	if err != nil {
		log.Printf("Warning: failed to get video info: %v", err)
		videoInfo = &models.VideoInfo{}
	}

	var renditions []*models.Rendition

	// Convert each quality to variant
	for _, variant := range models.ABRLadder {
		log.Printf("Converting %s variant...", variant.Name)

		outputFile := filepath.Join(outputDir, fmt.Sprintf("%s.m3u8", variant.Name))
		segmentationPattern := filepath.Join(outputDir, fmt.Sprintf("%s_%%03d.ts", variant.Name))

		cmd := exec.Command("ffmpeg",
			"-i", inputPath,
			"-vf", fmt.Sprintf("scale=%s", variant.Resolution),
			"-c:v", "libx264",
			"-b:v", variant.Bitrate,
			"-c:a", "aac",
			"-b:a", variant.AudioBitrate,
			"-hls_time", "10",
			"-hls_list_size", "0",
			"-hls_segment_filename", segmentationPattern,
			"-f", "hls",
			outputFile,
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Error converting %s: %v, output: %s", variant.Name, err, string(output))
			continue // Continue with other variants
		}

		log.Printf("Successfully converted %s", variant.Name)

		renditions = append(renditions, &models.Rendition{
			Name:       variant.Name,
			Resolution: variant.Resolution,
			Bitrate:    variant.Bitrate,
			Playlist:   fmt.Sprintf("%s.m3u8", variant.Name),
		})
	}

	if len(renditions) == 0 {
		return nil, "", nil, fmt.Errorf("failed to create any renditions")
	}

	// Create master playlist
	masterPlaylistPath := filepath.Join(outputDir, "master.m3u8")
	masterContent := "#EXTM3U\n#EXT-X-VERSION:3\n\n"

	for _, rendition := range renditions {
		// Extract bandwidth from bitrate (e.g., "5000k" -> 5000000)
		bitrateStr := strings.TrimSuffix(rendition.Bitrate, "k")
		bandwidth, _ := strconv.Atoi(bitrateStr)
		bandwidth *= 1000

		masterContent += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s\n", bandwidth, rendition.Resolution)
		masterContent += fmt.Sprintf("%s\n\n", rendition.Playlist)
	}

	if err := os.WriteFile(masterPlaylistPath, []byte(masterContent), 0644); err != nil {
		return nil, "", nil, fmt.Errorf("failed to write master playlist: %v", err)
	}

	log.Printf("Created master playlist at %s", masterPlaylistPath)

	return renditions, outputDir, videoInfo, nil
}

// getvideoinfo extracts metadata from video file using ffprobe
func getVideoInfo(videoPath string) (*models.VideoInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffprobe error: %v, output: %s", err, string(output))
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	info := &models.VideoInfo{
		Codec: "unknown",
	}

	// Parse duration
	if result.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
			info.Duration = duration
		}
	}

	// Find video stream
	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			info.Width = stream.Width
			info.Height = stream.Height
			info.Codec = stream.CodecName
			break
		}
	}

	return info, nil
}

// uploadToGCS uploads HLS files to Google Cloud Storage
func uploadToGCS(ctx context.Context, videoID, localDir string) (string, error) {
	if storageClient == nil {
		return "", fmt.Errorf("GCS client not initialized")
	}

	log.Printf("Uploading %s to GCS...", videoID)

	gcsPrefix := fmt.Sprintf("videos/%s/", videoID)

	// Walk through all files in the directory
	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Determine content type
		contentType := "video/MP2T"
		if strings.HasSuffix(path, ".m3u8") {
			contentType = "application/vnd.apple.mpegurl"
		}

		// Create blob name
		relPath, _ := filepath.Rel(localDir, path)
		blobName := gcsPrefix + relPath

		// Upload file
		obj := bucket.Object(blobName)
		writer := obj.NewWriter(ctx)
		writer.ContentType = contentType

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", path, err)
		}
		defer file.Close()

		if _, err := io.Copy(writer, file); err != nil {
			return fmt.Errorf("failed to upload %s: %v", blobName, err)
		}

		if err := writer.Close(); err != nil {
			return fmt.Errorf("failed to close writer for %s: %v", blobName, err)
		}

		log.Printf("Uploaded %s to %s", relPath, blobName)
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to GCS: %v", err)
	}

	masterPlaylistPath := gcsPrefix + "master.m3u8"
	return masterPlaylistPath, nil
}

// saveMetadata saves video metadata to GCS
func saveMetadata(ctx context.Context, videoID string, metadata *models.VideoMetadata) error {
	if storageClient == nil {
		return fmt.Errorf("GCS client not initialized")
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	blobName := fmt.Sprintf("metadata/%s.json", videoID)
	obj := bucket.Object(blobName)
	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"

	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write metadata: %v", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close metadata writer: %v", err)
	}

	return nil
}
