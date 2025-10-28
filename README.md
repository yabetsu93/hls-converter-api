# HLS Video Converter API

A production-ready Go service that converts MP4 videos to HLS (HTTP Live Streaming) format with adaptive bitrate (ABR) streaming, deployed on Google Cloud Platform.

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.24+ (for local development)
- Google Cloud Platform account with configured service account
- FFmpeg (installed in Docker container)

### Local Development

1. **Clone the repository**
```bash
git clone https://github.com/yabetsu93/hls-converter-api.git
cd hls-converter-api
```

2. **Set up Google Cloud credentials**
```bash
# Download your service account JSON key
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
export BUCKET_NAME=assignment-bucket
```

3. **Run with Docker Compose**
```bash
docker-compose up --build
```

The API will be available at `http://localhost:8080`

### Production Deployment

The service automatically deploys to Google Cloud Run on push to `main` branch via GitHub Actions.

**Prerequisites for CI/CD:**
- Set GitHub secrets:
  - `GCP_PROJECT_ID`: Your GCP project ID
  - `GCP_SA_KEY`: Service account JSON key (base64 encoded)

**Manual deployment:**
```bash
# Build and push image
docker build -t gcr.io/hls-converter-api/hls-converter-api .
docker push gcr.io/hls-converter-api/hls-converter-api

# Deploy to Cloud Run
gcloud run deploy hls-converter-api \
  --image gcr.io/hls-converter-api/hls-converter-api \
  --region asia-northeast1 \
  --platform managed \
  --allow-unauthenticated \
  --memory 2Gi \
  --cpu 2 \
  --timeout 3600 \
  --set-env-vars BUCKET_NAME=assignment-bucket \
  --service-account assignments@reddotdrone.jp
```

## 📋 API Documentation

### Base URL
- Local: `http://localhost:8080`
- Production: `https://hls-converter-api-<hash>.a.run.app`

### Endpoints

#### 1. Health Check
```http
GET /
```

**Response:**
```json
{
  "service": "HLS Video Converter API",
  "status": "running",
  "version": "1.0.0"
}
```

#### 2. Upload Video
```http
POST /videos
Content-Type: multipart/form-data
```

**Request:**
- `file`: MP4 video file (required)
- `title`: Custom title for the video (optional)

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "sample.mp4",
  "title": "My Sample Video",
  "status": "completed",
  "duration": 300.5,
  "width": 1920,
  "height": 1080,
  "created_at": "2025-10-28T10:30:00Z",
  "master_playlist_url": "https://storage.googleapis.com/...",
  "renditions": [
    {
      "name": "1080p",
      "resolution": "1920x1080",
      "bitrate": "5000k",
      "url": "https://storage.googleapis.com/..."
    },
    {
      "name": "720p",
      "resolution": "1280x720",
      "bitrate": "3000k",
      "url": "https://storage.googleapis.com/..."
    },
    {
      "name": "480p",
      "resolution": "854x480",
      "bitrate": "1500k",
      "url": "https://storage.googleapis.com/..."
    },
    {
      "name": "360p",
      "resolution": "640x360",
      "bitrate": "800k",
      "url": "https://storage.googleapis.com/..."
    }
  ]
}
```

#### 3. List Videos
```http
GET /videos?q=search&limit=10&offset=0
```

**Query Parameters:**
- `q`: Search query (searches in filename and title)
- `limit`: Number of results per page (default: 10, max: 100)
- `offset`: Pagination offset (default: 0)

**Response:**
```json
{
  "total": 42,
  "videos": [
    {
      "id": "...",
      "filename": "...",
      "title": "...",
      "status": "completed",
      "duration": 300.5,
      "created_at": "...",
      "master_playlist_url": "..."
    }
  ]
}
```

#### 4. Get Video Details
```http
GET /videos/:id
```

**Response:** Same as upload response with refreshed signed URLs

## 🧪 Testing

### Sample Video
A 5-minute sample MP4 video for testing is recommended. You can use:
- Create one with FFmpeg: `ffmpeg -f lavfi -i testsrc=duration=300:size=1920x1080:rate=30 -pix_fmt yuv420p sample.mp4`
- Download from: https://sample-videos.com/

### cURL Examples

**Upload a video:**
```bash
curl -X POST http://localhost:8080/videos \
  -F "file=@sample.mp4" \
  -F "title=Test Video"
```

**List all videos:**
```bash
curl http://localhost:8080/videos
```

**Search videos:**
```bash
curl "http://localhost:8080/videos?q=test&limit=5"
```

**Get video details:**
```bash
curl http://localhost:8080/videos/550e8400-e29b-41d4-a716-446655440000
```

### Postman Collection

Import the following into Postman:

```json
{
  "info": {
    "name": "HLS Converter API",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Health Check",
      "request": {
        "method": "GET",
        "url": "{{base_url}}/"
      }
    },
    {
      "name": "Upload Video",
      "request": {
        "method": "POST",
        "url": "{{base_url}}/videos",
        "body": {
          "mode": "formdata",
          "formdata": [
            {
              "key": "file",
              "type": "file",
              "src": "/path/to/sample.mp4"
            },
            {
              "key": "title",
              "value": "Test Video",
              "type": "text"
            }
          ]
        }
      }
    },
    {
      "name": "List Videos",
      "request": {
        "method": "GET",
        "url": "{{base_url}}/videos?limit=10&offset=0"
      }
    },
    {
      "name": "Search Videos",
      "request": {
        "method": "GET",
        "url": "{{base_url}}/videos?q=test"
      }
    },
    {
      "name": "Get Video",
      "request": {
        "method": "GET",
        "url": "{{base_url}}/videos/:id"
      }
    }
  ],
  "variable": [
    {
      "key": "base_url",
      "value": "http://localhost:8080"
    }
  ]
}
```

### Testing HLS Playback

Once converted, you can test HLS playback using:

1. **VLC Media Player:**
   - Open the master playlist URL in VLC
   
2. **HLS.js demo player:**
   ```html
   <!DOCTYPE html>
   <html>
   <head>
     <script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
   </head>
   <body>
     <video id="video" controls width="640"></video>
     <script>
       var video = document.getElementById('video');
       var videoSrc = 'YOUR_MASTER_PLAYLIST_URL';
       if (Hls.isSupported()) {
         var hls = new Hls();
         hls.loadSource(videoSrc);
         hls.attachMedia(video);
       }
     </script>
   </body>
   </html>
   ```

3. **FFplay:**
   ```bash
   ffplay "https://storage.googleapis.com/.../master.m3u8"
   ```

## 🏗️ Architecture & Design Decisions

### Technology Stack

**Go + Gin Framework:**
- **Why Go:** High performance, excellent concurrency support for video processing, small binary size for containers
- **Why Gin:** Fast HTTP router, middleware support, good documentation
- **Alternatives considered:** Node.js (slower for CPU-intensive tasks), Python FastAPI (GIL limitations)

### Video Processing Pipeline

**FFmpeg for HLS Conversion:**
- Industry standard for video processing
- Native HLS support with segmentation
- Hardware acceleration support (future enhancement)

**ABR Ladder (4 quality levels):**
```
1080p: 1920x1080 @ 5000kbps (high-end devices, fast connections)
720p:  1280x720  @ 3000kbps (standard quality)
480p:  854x480   @ 1500kbps (mobile devices)
360p:  640x360   @ 800kbps  (slow connections)
```

**Design decisions:**
- 10-second segments for balance between latency and overhead
- H.264 codec for maximum compatibility
- AAC audio for broad device support
- Separate playlists per quality level + master playlist

### Storage Strategy

**Google Cloud Storage (GCS):**
- Durable, scalable object storage
- Native integration with Cloud Run
- Cost-effective for video files
- Signed URLs for secure, temporary access (60-minute expiration)

**Storage structure:**
```
gs://assignment-bucket/
├── videos/
│   └── {video-id}/
│       ├── master.m3u8
│       ├── 1080p.m3u8
│       ├── 1080p_000.ts
│       ├── 1080p_001.ts
│       ├── 720p.m3u8
│       └── ...
└── metadata/
    └── {video-id}.json
```

**Why signed URLs:**
- Security: Temporary access without making bucket public
- Flexibility: Can add authentication later
- Trade-off: URLs expire (60 min) - acceptable for this use case

### API Design

**RESTful principles:**
- Resource-based URLs (`/videos`, `/videos/:id`)
- Standard HTTP methods (GET, POST)
- JSON responses
- Proper status codes

**Search & Pagination:**
- Simple text search in filename/title (sufficient for MVP)
- Cursor-based pagination could be added for large datasets
- Trade-off: In-memory filtering (acceptable for < 10K videos)

### Deployment & Infrastructure

**Cloud Run:**
- **Why:** Serverless, auto-scaling, pay-per-use
- **Why not Compute Engine:** Over-provisioning, manual scaling
- **Why not Kubernetes:** Overkill for single service, higher complexity

**Resource allocation:**
```
Memory: 2GB (video processing is memory-intensive)
CPU: 2 cores (parallel FFmpeg encoding)
Timeout: 3600s (1 hour for large videos)
Max instances: 10 (cost control)
```

**CI/CD Pipeline:**
- GitHub Actions for automation
- Build → Test → Push to Artifact Registry → Deploy to Cloud Run
- Triggers on push to `main`

### Concurrency & Performance

**Current approach:**
- Synchronous processing (simpler, acceptable for MVP)
- One video per request
- Multiple quality levels processed sequentially

**Future improvements:**
- Background job queue (Cloud Tasks, Pub/Sub)
- Parallel rendition encoding
- Chunked upload for large files
- Progress tracking via WebSocket or polling

### Error Handling

**Strategy:**
- Graceful degradation: If one quality level fails, others continue
- Detailed logging for debugging
- Proper HTTP status codes
- Cleanup of temporary files on error

### Security Considerations

**Current:**
- Signed URLs for GCS access
- File type validation (MP4 only)
- CORS headers for browser access
- Service account with minimal permissions

**Production hardening needed:**
- Rate limiting
- File size limits
- Input validation & sanitization
- Virus scanning for uploaded files
- Authentication & authorization
- Content security policy headers

## 📊 Monitoring & Observability

**Logging:**
- Structured logging to stdout (captured by Cloud Run)
- Key events: upload, conversion start/end, errors

**Metrics (Cloud Run provides):**
- Request count, latency, errors
- CPU & memory utilization
- Container instance count

**Future improvements:**
- Custom metrics (conversion duration, file sizes)
- Distributed tracing (OpenTelemetry)
- Alerting (error rate, latency spikes)

## 🚧 Known Limitations & Future Work

### Current Limitations

1. **Synchronous processing:** Uploads timeout on large videos (>1GB)
2. **No progress tracking:** Client can't monitor conversion status
3. **Fixed ABR ladder:** Doesn't adapt to source video quality
4. **Memory constraints:** Large videos may cause OOM
5. **No authentication:** Anyone can upload videos
6. **Simple search:** No filtering by duration, resolution, bitrate yet
7. **Signed URL expiration:** URLs expire after 60 minutes

### What I'd Do with Another Day

#### High Priority (Day 2)

1. **Async Job Processing:**
   - Implement Cloud Tasks for background processing
   - Add job status tracking (queued, processing, completed, failed)
   - Return job ID immediately, poll for status
   ```go
   POST /videos -> {job_id: "...", status: "queued"}
   GET /jobs/:id -> {status: "processing", progress: 45}
   ```

2. **Advanced Search & Filtering:**
   ```go
   GET /videos?duration_min=60&duration_max=600
   GET /videos?resolution=1920x1080
   GET /videos?bitrate_min=3000
   ```

3. **Streaming Upload:**
   - Accept GCS URI directly (no local storage)
   - Chunked multipart upload for large files

4. **Progress WebSocket:**
   ```go
   ws://api/videos/:id/progress
   -> {stage: "encoding_720p", percent: 67}
   ```

#### Medium Priority (Week 1)

5. **Authentication & Authorization:**
   - JWT tokens or API keys
   - User-scoped video access
   - Rate limiting per user

6. **Adaptive ABR Ladder:**
   - Analyze source video quality
   - Generate only necessary renditions
   - Skip upscaling if source is 720p

7. **Enhanced Metadata:**
   - Extract thumbnails
   - Scene detection
   - Audio/subtitle track info

8. **Batch Operations:**
   ```go
   POST /videos/batch -> Upload multiple videos
   DELETE /videos/batch -> Delete multiple videos
   ```

#### Lower Priority (Month 1)

9. **Cost Optimization:**
   - Use Coldline storage for old videos
   - Lifecycle policies for auto-deletion
   - Compression before storage

10. **Advanced Features:**
    - Watermarking
    - DRM support
    - Live streaming support
    - Thumbnail generation at intervals

11. **Performance:**
    - GPU acceleration for encoding
    - CDN integration for delivery
    - Multi-region deployment

12. **Developer Experience:**
    - SDK clients (Go, Python, JS)
    - GraphQL API
    - Webhook notifications

## 🔧 Configuration

### Environment Variables

```bash
# Required
BUCKET_NAME=assignment-bucket              # GCS bucket name
GOOGLE_APPLICATION_CREDENTIALS=path/to/sa.json  # Service account key

# Optional
PORT=8080                                  # Server port (default: 8080)
GIN_MODE=release                          # Gin mode: debug/release
```

### FFmpeg Customization

Edit the ABR ladder in `main.go`:

```go
var abrLadder = []struct {
    Name         string
    Resolution   string
    Bitrate      string
    AudioBitrate string
}{
    {"4K", "3840x2160", "15000k", "256k"},  // Add 4K
    {"1080p", "1920x1080", "5000k", "192k"},
    // ... rest of ladder
}
```

## 📝 Development Notes

### Time Breakdown (4 hours)

- **Architecture & Design:** 30 min
- **Core Video Conversion:** 1 hour
- **API Implementation:** 1 hour
- **GCS Integration:** 45 min
- **Docker & CI/CD:** 30 min
- **Testing & Documentation:** 45 min

### Trade-offs Made

1. **Synchronous vs Async:** Chose sync for simplicity, knowing it's not production-ready for large files
2. **In-memory search:** Fast for MVP, not scalable (should use database)
3. **Fixed ABR ladder:** Simple but wastes storage on low-quality sources
4. **No database:** Uses GCS metadata files - simpler but limited query capability
5. **60-min signed URLs:** Balance between security and UX

## 📄 License

MIT License - feel free to use for your projects!

## 👥 Contributing

Contributions welcome! Areas that need help:
- Async processing implementation
- Advanced search with filters
- Authentication system
- Performance optimizations
- Test coverage

## 🆘 Support

For issues or questions:
- Check logs: `gcloud run logs read hls-converter-api`
- GCP Console: Cloud Run service logs
- GitHub Issues: [Create an issue]

---

**Built with ❤️ for high-quality video streaming**
