# Torrentium Web Share Portal

A web service for publishing and sharing Torrentium file magnet links through a browsable public portal.

## Features

- 📡 **Easy Publishing**: Publish files directly from the Torrentium app with a single click
- 🔍 **Search & Browse**: Find shared files by name, description, tags, or category
- 🔗 **Magnet Links**: Generate `torrentium://` magnet links for easy sharing
- 🏷️ **Categories & Tags**: Organize files with categories and custom tags
- 👁️ **Privacy Controls**: Public (searchable) or Unlisted (link-only) visibility
- ⏰ **Expiration**: Optional expiration dates for temporary shares
- 🚩 **Moderation**: Built-in reporting system for inappropriate content
- 📊 **Statistics**: Track downloads and views for your published files

## Quick Start

### Running Locally

```bash
cd webshare

# Install dependencies
go mod tidy
cd frontend && npm install && npm run build && cd ..

# Run the server
go run main.go
```

The portal will be available at `http://localhost:8080/portal/`

### Configuration

Environment variables:
- `PORT` - Server port (default: 8080)
- `WEBSHARE_DB` - Database path (default: ./webshare.db)
- `CORS_ORIGINS` - Allowed CORS origins (default: *)
- `API_KEY` - Optional API key for publishing protection

Command line flags:
```bash
go run main.go -port 3000 -db ./data/webshare.db -api-key mysecretkey
```

## API Endpoints

### Public Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/files` | List published files (paginated) |
| GET | `/api/v1/files/:cid` | Get file details by CID |
| GET | `/api/v1/categories` | Get available categories |
| GET | `/api/v1/stats` | Get portal statistics |
| POST | `/api/v1/files/:cid/download` | Track download event |
| POST | `/api/v1/files/:id/report` | Report a file |

### Publishing Endpoints (Protected by API Key if configured)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/publish` | Publish a file |
| DELETE | `/api/v1/publish/:cid` | Unpublish a file |

### Query Parameters for `/api/v1/files`

- `page` - Page number (default: 1)
- `limit` - Items per page (default: 20, max: 100)
- `category` - Filter by category
- `search` - Search in filename, description, tags
- `sort` - Sort order: `newest`, `oldest`, `downloads`, `views`, `size`, `name`

### Publish Request Body

```json
{
  "cid": "bafybeihash...",
  "filename": "example.zip",
  "fileSize": 1024000,
  "description": "Optional description",
  "category": "archives",
  "tags": ["tag1", "tag2"],
  "visibility": "public",
  "publisherId": "12D3KooW...",
  "expiresIn": 168
}
```

## Magnet Link Format

Torrentium magnet links follow this format:
```
torrentium://<cid>?dn=<filename>&sz=<size>
```

Example:
```
torrentium://bafybeihash...?dn=MyFile.zip&sz=1048576
```

## Deployment

### Docker

```bash
docker build -t torrentium-webshare .
docker run -p 8080:8080 -v ./data:/app/data torrentium-webshare
```

### Railway/Render

1. Connect your repository
2. Set the root directory to `webshare/`
3. Build command: `go build -o server . && cd frontend && npm install && npm run build`
4. Start command: `./server`
5. Add environment variables as needed

## Integration with Torrentium App

The Torrentium desktop app includes built-in support for publishing to the web portal:

1. Click the "🌐" button on any file in the Files view
2. Fill in optional details (description, category, tags)
3. Choose visibility (public or unlisted)
4. Click "Publish"

Configure the portal URL in Settings > Web Share:
- **Portal URL**: Your deployed portal URL (e.g., `https://share.torrentium.io`)
- **API Key**: If your portal requires authentication

## Self-Hosting

You can self-host your own portal instance for private sharing:

1. Clone this repository
2. Build and run the webshare server
3. Configure the Torrentium app to point to your server
4. Share the portal URL with your network

## License

Part of the Torrentium project. See the main repository for license information.
