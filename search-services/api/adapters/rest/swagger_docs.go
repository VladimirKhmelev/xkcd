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
	Total  int          `json:"total" example:"5"`
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
// @Description  Search XKCD comics by phrase using full-text search against the database
// @Tags         search
// @Produce      json
// @Param        phrase query string true  "Search phrase"                      minlength(1)
// @Param        limit  query int    false "Max number of results" default(10)  minimum(1)  maximum(100)
// @Success      200    {object} SearchResponse
// @Failure      400    {string} string "phrase is required"
// @Failure      500    {string} string "internal server error"
// @Router       /api/search [get]
func Search() {}

// ISearch godoc
// @Summary      Index search
// @Description  Search XKCD comics using in-memory index, faster but requires index to be built
// @Tags         search
// @Produce      json
// @Param        phrase query string true  "Search phrase"                      minlength(1)
// @Param        limit  query int    false "Max number of results" default(10)  minimum(1)  maximum(100)
// @Success      200    {object} SearchResponse
// @Failure      400    {string} string "phrase is required"
// @Failure      500    {string} string "internal server error"
// @Router       /api/isearch [get]
func ISearch() {}

// DBUpdate godoc
// @Summary      Update database
// @Description  Fetch new comics from xkcd.com and store them. Runs asynchronously — returns 202 if already running
// @Tags         admin
// @Security     BearerAuth
// @Success      200 {string} string "ok"
// @Success      202 {string} string "already running"
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
