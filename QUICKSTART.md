# Quick Start Guide

Get the HLS Video Converter API running in 5 minutes!

## Prerequisites

- Docker installed
- Google Cloud account (for production)
- 5-minute sample video (or create one)

## Local Development (2 minutes)

### 1. Clone and Run

```bash
# Clone repository
git clone https://github.com/yabetsu93/hls-converter-api.git
cd hls-converter-api

# Create sample video (optional)
make create-sample-video

# Run with Docker Compose
docker-compose up --build
```

### 2. Test the API

```bash
# Health check
curl http://localhost:8080/

# Upload video
curl -X POST http://localhost:8080/videos \
  -F "file=@sample.mp4" \
  -F "title=My Test Video"

# List videos
curl http://localhost:8080/videos
```

**That's it!** Your API is running locally.

---

## Production Deployment (3 minutes)

### 1. Setup GCP

```bash
# Set your project
export GCP_PROJECT_ID=your-project-id

# Authenticate
gcloud auth login
gcloud config set project $GCP_PROJECT_ID

# Create Artifact Registry (one-time setup)
gcloud artifacts repositories create hls-converter \
  --repository-format=docker \
  --location=asia-northeast1
```

### 2. Deploy

```bash
# Run deployment script
./deploy.sh $GCP_PROJECT_ID asia-northeast1
```

### 3. Test Production

```bash
# Get your service URL
SERVICE_URL=$(gcloud run services describe hls-converter-api \
  --region asia-northeast1 \
  --format 'value(status.url)')

# Test it
curl $SERVICE_URL/
```

**Done!** Your API is live on Cloud Run.

---

## CI/CD Setup (2 minutes)

### 1. Add GitHub Secrets

Go to your GitHub repository → Settings → Secrets → Actions:

- `GCP_PROJECT_ID`: Your GCP project ID
- `GCP_SA_KEY`: Service account JSON (base64 encoded)

### 2. Push to Main

```bash
git add .
git commit -m "Initial deployment"
git push origin main
```

**Automatic deployment** will trigger on every push to `main`.

---

## Quick API Reference

### Upload Video
```bash
curl -X POST $BASE_URL/videos \
  -F "file=@video.mp4" \
  -F "title=My Video"
```

### List Videos
```bash
curl "$BASE_URL/videos?limit=10"
```

### Search Videos
```bash
curl "$BASE_URL/videos?q=test"
```

### Get Video Details
```bash
curl "$BASE_URL/videos/{video-id}"
```

---

## Troubleshooting

**Port already in use?**
```bash
# Change port in docker-compose.yml
ports:
  - "8081:8080"  # Change 8080 to 8081
```

**Docker build fails?**
```bash
# Clean and rebuild
docker-compose down
docker system prune -a
docker-compose up --build
```

**GCP authentication error?**
```bash
# Re-authenticate
gcloud auth login
gcloud auth application-default login
```

---

## Next Steps

- 📖 Read the full [README.md](README.md)
- 🏗️ See [ARCHITECTURE.md](ARCHITECTURE.md) for design details
- 🤝 Check [CONTRIBUTING.md](CONTRIBUTING.md) to contribute
- 📮 Import [postman_collection.json](postman_collection.json) for API testing

---

## Support

- Issues: [GitHub Issues](link)
- Logs: `gcloud run logs tail hls-converter-api`
- Docs: [Cloud Run Documentation](https://cloud.google.com/run/docs)

**Happy coding! 🚀**
