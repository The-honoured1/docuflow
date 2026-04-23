# 📄 DocuFlow

> A lightweight, server-rendered documentation platform designed for teams that want clarity, speed, and control over their technical knowledge.

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![SQLite](https://img.shields.io/badge/SQLite-3-003B57?style=flat&logo=sqlite)](https://www.sqlite.org/)
[![HTMX](https://img.shields.io/badge/HTMX-1.9-3D72D7?style=flat&logo=htmx)](https://htmx.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## ✨ Features

DocuFlow delivers a rich collaborative documentation experience while staying simple, backend-driven, and production-ready.

### 🚀 Core Capabilities

- **📝 Markdown-First Editing** - Write documentation using familiar Markdown syntax with live preview
- **💾 Autosave** - HTMX-powered autosave every 2 seconds while editing
- **🔄 Version Control** - Complete revision history with one-click rollback to any previous version
- **📎 File Attachments** - Upload and manage files (PDF, images, docs) directly on documents
- **🔒 Password Protection** - Secure document access with bcrypt-hashed passwords
- **🔗 Shareable Links** - Generate unique share tokens for public (or password-gated) document access
- **💬 Inline Comments** - Threaded discussions directly on documents
- **🔍 Full-Text Search** - Fast search across all document titles and content
- **👥 User Authentication** - Secure registration and login with bcrypt password hashing
- **🎨 Beautiful UI** - Industry-standard design with Inter font, modern shadows, and smooth animations
- **📱 Responsive Design** - Works seamlessly on desktop, tablet, and mobile devices

### 🏗️ Technical Highlights

- **Server-Rendered** - No SPA complexity, just clean HTML from the server
- **Zero JavaScript Framework** - Uses HTMX for dynamic interactions
- **Pure Go SQLite Driver** - No CGO required (`modernc.org/sqlite`)
- **Minimal Dependencies** - Only 3 external packages needed
- **Fast & Lightweight** - Optimized for low-resource deployments

## 🎯 Why DocuFlow?

DocuFlow demonstrates how modern web applications can deliver rich user experiences while staying simple, backend-driven, and production-ready. Perfect for:

- 📚 Internal documentation and wikis
- 🛠️ Engineering knowledge bases
- 📋 Product specifications
- 🚀 Startup documentation hubs
- 📖 Technical writing teams

Where **correctness**, **auditability**, and **maintainability** matter more than flashy UI frameworks.

## 🚀 Quick Start

### Prerequisites

- Go 1.25 or higher
- Internet connection (for initial dependency download)

### Installation

```bash
# Clone the repository
git clone https://github.com/The-honoured1/docuflow.git
cd docuflow

# Install dependencies
go mod tidy

# Run the server
go run main.go
```

The server will start on `http://localhost:8080`

### First Steps

1. **Register an account** at `http://localhost:8080/register`
2. **Create your first document** using the "New Document" button
3. **Write in Markdown** - Use headings, lists, code blocks, and more
4. **Collaborate** - Add comments and track changes

## 📁 Project Structure

```
docuflow/
├── db/
│   └── db.go                # Database initialization & schema
├── handlers/
│   ├── auth.go              # Authentication (register, login, logout)
│   ├── document.go          # Document CRUD & Markdown rendering
│   ├── revision.go          # Version history & rollback
│   ├── comment.go           # Inline comments
│   ├── upload.go            # File upload workflow
│   └── search.go            # Full-text search
├── models/
│   └── models.go            # Data structures
├── web/
│   ├── static/
│   │   └── css/
│   │       └── style.css        # Premium design system
│   └── templates/
│       ├── base.html            # Base layout
│       ├── document_*.html      # Document views
│       ├── share_view.html      # Public share view
│       ├── login.html           # Authentication
│       ├── register.html
│       ├── search.html 
│       ├── revisions.html
│       └── partials/
│           └── comments.html    # Comment component
├── .github/
│   └── workflows/           # CI/CD (CI, Release)
├── Dockerfile               # Containerization
├── .gitignore               # Root level ignore
├── main.go                  # Application entry point
├── go.mod                   # Go dependencies
├── go.sum
└── README.md
```

## 🛠️ Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Backend** | Go 1.25+ | Fast, compiled server-side logic |
| **Database** | SQLite | Embedded, zero-config database |
| **Templates** | Go html/template | Server-side HTML rendering |
| **Interactivity** | HTMX 1.9 | Dynamic updates without JavaScript |
| **Markdown** | gomarkdown | Markdown to HTML conversion |
| **Auth** | bcrypt | Secure password hashing |
| **Styling** | Vanilla CSS | Modern design system with Inter font |

## 🎨 Design System

DocuFlow features a premium, industry-standard design system:

- **Typography**: Inter font family for UI, JetBrains Mono for code
- **Color Palette**: Comprehensive 50-900 shade system
- **Components**: Buttons, cards, forms, badges, and more
- **Shadows**: Layered elevation system
- **Animations**: Smooth transitions and micro-interactions
- **Responsive**: Mobile-first, works on all screen sizes

## 🔐 Security Features

- **Password Hashing**: bcrypt with default cost factor
- **SQL Injection Protection**: Parameterized queries throughout
- **Session Management**: HTTP-only cookies
- **Input Validation**: Server-side validation on all forms

## 🚀 Deployment

### CI/CD with GitHub Actions

DocuFlow includes two pre-configured workflows:
- **CI**: Runs on every push to `main` to build and vet code.
- **Release**: Triggered by version tags (e.g., `v1.0.0`). Compiles binaries for Linux, Windows, and macOS and publishes them to a GitHub Release.

### Docker

Build and run using the optimized multi-stage Dockerfile:

```bash
# Build
docker build -t docuflow .

# Run
docker run -p 8080:8080 -v $(pwd)/data:/app/uploads docuflow
```

## 📝 API Routes

| Method | Route | Description |
|--------|-------|-------------|
| `GET` | `/` | Document list (home) |
| `GET/POST` | `/register` | User registration |
| `GET/POST` | `/login` | User login |
| `GET` | `/logout` | User logout |
| `GET/POST` | `/documents/new` | Create document |
| `GET` | `/documents/view?id={id}` | View document |
| `GET/POST` | `/documents/edit?id={id}` | Edit document |
| `POST` | `/documents/autosave` | Autosave (HTMX) |
| `POST` | `/documents/set-password` | Set document protection |
| `POST` | `/documents/share` | Generate/Revoke share link |
| `GET` | `/share/{token}` | Public document view |
| `POST` | `/documents/upload` | Upload attachment |
| `POST` | `/documents/delete-file` | Remove attachment |
| `GET` | `/files/download` | Download attachment |
| `GET` | `/revisions?doc_id={id}` | Revision history |
| `GET` | `/revisions/view?id={id}` | View revision |
| `POST` | `/revisions/rollback` | Restore revision |
| `GET` | `/comments?doc_id={id}` | List comments (HTMX) |
| `POST` | `/comments/add` | Add comment (HTMX) |
| `POST` | `/comments/delete` | Delete comment (HTMX) |
| `GET` | `/search?q={query}` | Search documents |

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [HTMX](https://htmx.org/) - For making server-side rendering interactive
- [gomarkdown](https://github.com/gomarkdown/markdown) - For excellent Markdown parsing
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) - For pure Go SQLite driver
- [Inter Font](https://rsms.me/inter/) - For beautiful typography

## 📧 Contact

For questions or feedback, please open an issue on GitHub.

---

**Built with ❤️ using Go, SQLite, and HTMX**

*DocuFlow - Where documentation meets simplicity*
