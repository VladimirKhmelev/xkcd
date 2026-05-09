package rest

// LoginRequest represents login credentials
type LoginRequest struct {
	Name     string `json:"name" example:"admin"`
	Password string `json:"password" example:"password"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Comics []ComicsItem `json:"comics"`
	Total  int          `json:"total" example:"10"`
}

// ComicsItem represents a single comic
type ComicsItem struct {
	ID  int    `json:"id" example:"353"`
	URL string `json:"url" example:"https://imgs.xkcd.com/comics/python.png"`
}

// AdminLogin godoc
// @Summary      Admin login
// @Description  Login with admin credentials, returns JWT token
// @Tags         auth
// @Accept       json
// @Produce      plain
// @Param        body body LoginRequest true "Credentials"
// @Success      200 {string} string "JWT token"
// @Failure      401 {string} string "unauthorized"
// @Router       /api/login [post]
func AdminLogin() {}

// UserLogin godoc
// @Summary      User login
// @Description  Login with registered user credentials, returns JWT token
// @Tags         auth
// @Accept       json
// @Produce      plain
// @Param        body body LoginRequest true "Credentials"
// @Success      200 {string} string "JWT token"
// @Failure      401 {string} string "unauthorized"
// @Router       /api/user/login [post]
func UserLogin() {}

// Register godoc
// @Summary      Register user
// @Description  Create a new user account
// @Tags         auth
// @Accept       json
// @Produce      plain
// @Param        body body LoginRequest true "Credentials"
// @Success      201 {string} string "created"
// @Failure      400 {string} string "name and password are required"
// @Failure      409 {string} string "user already exists"
// @Router       /api/register [post]
func Register() {}

// Search godoc
// @Summary      Search comics
// @Description  Search XKCD comics by phrase
// @Tags         search
// @Produce      json
// @Param        phrase query string true  "Search phrase"
// @Param        limit  query int    false "Result limit" default(10)
// @Success      200 {object} SearchResponse
// @Failure      400 {string} string "phrase is required"
// @Router       /api/search [get]
func Search() {}

// ISearch godoc
// @Summary      Index search
// @Description  Search XKCD comics using in-memory index
// @Tags         search
// @Produce      json
// @Param        phrase query string true  "Search phrase"
// @Param        limit  query int    false "Result limit" default(10)
// @Success      200 {object} SearchResponse
// @Failure      400 {string} string "phrase is required"
// @Router       /api/isearch [get]
func ISearch() {}

// DBUpdate godoc
// @Summary      Update database
// @Description  Fetch new comics from XKCD and store them
// @Tags         admin
// @Security     BearerAuth
// @Success      200 {string} string "ok"
// @Success      202 {string} string "already running"
// @Failure      401 {string} string "unauthorized"
// @Router       /api/db/update [post]
func DBUpdate() {}

// DBStats godoc
// @Summary      Database stats
// @Description  Returns comics and words counts
// @Tags         admin
// @Produce      json
// @Success      200 {object} map[string]int
// @Router       /api/db/stats [get]
func DBStats() {}

// DBStatus godoc
// @Summary      Update status
// @Description  Returns current update status (idle/running)
// @Tags         admin
// @Produce      json
// @Success      200 {object} map[string]string
// @Router       /api/db/status [get]
func DBStatus() {}

// DBDrop godoc
// @Summary      Drop database
// @Description  Delete all comics from the database
// @Tags         admin
// @Security     BearerAuth
// @Success      200 {string} string "ok"
// @Failure      401 {string} string "unauthorized"
// @Router       /api/db [delete]
func DBDrop() {}
