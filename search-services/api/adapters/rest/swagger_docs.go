package rest

// LoginRequest represents login credentials
// @Description Login credentials for admin or user
type LoginRequest struct {
	Name     string `json:"name"     example:"admin"    minLength:"1"`
	Password string `json:"password" example:"password" minLength:"1"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Comics []ComicsItem `json:"comics"`
	Total  int          `json:"total" example:"52"`
	Page   int          `json:"page"  example:"1"`
	Pages  int          `json:"pages" example:"5"`
}

// ISearchResponse represents index search results
type ISearchResponse struct {
	Comics []ComicsItem `json:"comics"`
	Total  int          `json:"total" example:"7"`
}

// PingResponse represents health check results
type PingResponse struct {
	Replies map[string]string `json:"replies" example:"search:ok,update:ok,words:ok"`
}

// ComicsItem represents a single comic
type ComicsItem struct {
	ID  int    `json:"id"  example:"353"`
	URL string `json:"url" example:"https://imgs.xkcd.com/comics/python.png"`
}

// StatsResponse represents database statistics
type StatsResponse struct {
	WordsTotal    int `json:"words_total"    example:"148234"`
	WordsUnique   int `json:"words_unique"   example:"9412"`
	ComicsFetched int `json:"comics_fetched" example:"3241"`
	ComicsTotal   int `json:"comics_total"   example:"3241"`
}

// StatusResponse represents current update status
type StatusResponse struct {
	Status string `json:"status" example:"idle" enums:"idle,running"`
}

// Ping godoc
// @Summary      Health check
// @Description  Checks connectivity to all internal services (search, update, words)
// @Tags         system
// @Produce      json
// @Success      200 {object} PingResponse
// @Router       /api/ping [get]
func Ping() {}

// AdminLogin godoc
// @Summary      Admin login
// @Description  Login with admin credentials (from env ADMIN_USER/ADMIN_PASSWORD), returns JWT token
// @Tags         auth
// @Accept       json
// @Produce      plain
// @Param        body body LoginRequest true "Admin credentials"
// @Success      200 {string} string "eyJhbGciOiJIUzI1NiJ9..."
// @Failure      400 {string} string "invalid request body"
// @Failure      401 {string} string "unauthorized"
// @Router       /api/login [post]
func AdminLogin() {}

// UserLogin godoc
// @Summary      User login
// @Description  Login with registered user credentials, returns JWT token valid for searching
// @Tags         auth
// @Accept       json
// @Produce      plain
// @Param        body body LoginRequest true "User credentials"
// @Success      200 {string} string "eyJhbGciOiJIUzI1NiJ9..."
// @Failure      400 {string} string "invalid request body"
// @Failure      401 {string} string "unauthorized"
// @Router       /api/user/login [post]
func UserLogin() {}

// Register godoc
// @Summary      Register user
// @Description  Create a new user account with bcrypt-hashed password
// @Tags         auth
// @Accept       json
// @Produce      plain
// @Param        body body LoginRequest true "New user credentials"
// @Success      201 {string} string "created"
// @Failure      400 {string} string "name and password are required"
// @Failure      409 {string} string "user already exists or internal error"
// @Router       /api/register [post]
func Register() {}

// Search godoc
// @Summary      Search comics
// @Description  Full-text search against the database. Results are paginated — use page and limit to navigate. Returns 503 when concurrency limit (10 simultaneous requests) is reached.
// @Tags         search
// @Produce      json
// @Param        phrase query string true  "Search phrase"                     minlength(1)
// @Param        limit  query int    false "Results per page"  default(10)     minimum(1)  maximum(100)
// @Param        page   query int    false "Page number"       default(1)      minimum(1)
// @Success      200    {object} SearchResponse
// @Failure      400    {string} string "phrase is required / invalid limit / invalid page"
// @Failure      500    {string} string "internal server error"
// @Failure      503    {string} string "service unavailable (concurrency limit reached)"
// @Router       /api/search [get]
func Search() {}

// ISearch godoc
// @Summary      Index search
// @Description  Searches the in-memory index — faster than /api/search but requires the index to be built first (happens automatically after DB update). Limited to 100 rps; returns 503 when exceeded.
// @Tags         search
// @Produce      json
// @Param        phrase query string true  "Search phrase"                     minlength(1)
// @Param        limit  query int    false "Max number of results" default(10)  minimum(1)  maximum(100)
// @Success      200    {object} ISearchResponse
// @Failure      400    {string} string "phrase is required / invalid limit"
// @Failure      500    {string} string "internal server error"
// @Failure      503    {string} string "service unavailable (rate limit exceeded)"
// @Router       /api/isearch [get]
func ISearch() {}

// DBUpdate godoc
// @Summary      Update database
// @Description  Fetch missing comics from xkcd.com and store them. Blocking — returns 200 when done. Returns 202 immediately if an update is already in progress.
// @Tags         admin
// @Security     BearerAuth
// @Success      200 {string} string "ok"
// @Success      202 {string} string "update already in progress"
// @Failure      401 {string} string "unauthorized"
// @Failure      500 {string} string "internal server error"
// @Router       /api/db/update [post]
func DBUpdate() {}

// DBStats godoc
// @Summary      Database stats
// @Description  Returns total and fetched comics count, total and unique words count
// @Tags         admin
// @Produce      json
// @Success      200 {object} StatsResponse
// @Failure      500 {string} string "internal server error"
// @Router       /api/db/stats [get]
func DBStats() {}

// DBStatus godoc
// @Summary      Update status
// @Description  Returns current update status: idle or running
// @Tags         admin
// @Produce      json
// @Success      200 {object} StatusResponse
// @Failure      500 {string} string "internal server error"
// @Router       /api/db/status [get]
func DBStatus() {}

// DBDrop godoc
// @Summary      Drop database
// @Description  Delete all comics and words from the database, resets the search index and cache
// @Tags         admin
// @Security     BearerAuth
// @Success      200 {string} string "ok"
// @Failure      401 {string} string "unauthorized"
// @Failure      500 {string} string "internal server error"
// @Router       /api/db [delete]
func DBDrop() {}
