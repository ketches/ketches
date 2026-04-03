package handlers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func SignUp(c *gin.Context) {
	var req models.SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := services.SignUp(&req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrPublicSignUpDisabled) {
			status = http.StatusForbidden
		}
		api.Error(c, status, err)
		return
	}

	api.Created(c, toUserResponse(user))
}

func SignIn(c *gin.Context) {
	var req models.SignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	user, mustChangePassword, err := services.SignIn(&req)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, services.ErrAccountLocked) {
			status = http.StatusForbidden
		}
		api.Error(c, status, err)
		return
	}

	if err := issueUserSession(c, user, mustChangePassword); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.SignInResponse{
		User:               toUserResponse(user),
		MustChangePassword: mustChangePassword,
		DefaultPasswordNotice: func() string {
			if mustChangePassword {
				return "You are using the default administrator credentials. Please change the password as soon as possible."
			}
			return ""
		}(),
	})
}

func ListUsers(c *gin.Context) {
	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, users, err := services.ListUsers(req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	var res []models.UserResponse
	for _, u := range users {
		res = append(res, toUserResponse(&u))
	}

	api.Success(c, models.ListUsersResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Users:    res,
	})
}

func UpdateUser(c *gin.Context) {
	userID := c.Param("userID")
	var req struct {
		Fullname string `json:"fullname"`
		Email    string `json:"email"`
		Bio      string `json:"bio"`
		Phone    string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := services.UpdateUser(userID, req.Fullname, req.Email, req.Bio, req.Phone)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			status = http.StatusBadRequest
		}
		api.Error(c, status, err)
		return
	}

	api.Success(c, toUserResponse(user))
}

func GetCurrentUserProfile(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	user, err := services.GetCurrentUserProfile(claims.UserID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toUserResponse(user))
}

func UpdateCurrentUserProfile(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	var req models.UpdateCurrentUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := services.UpdateCurrentUserProfile(claims.UserID, req.Fullname, req.Email, req.Bio)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			status = http.StatusBadRequest
		}
		api.Error(c, status, err)
		return
	}

	api.Success(c, toUserResponse(user))
}

func ChangeCurrentUserPassword(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	var req models.ChangeCurrentUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.ChangeCurrentUserPassword(claims.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrInvalidCurrentPassword) || errors.Is(err, services.ErrWeakPassword) {
			status = http.StatusBadRequest
		}
		api.Error(c, status, err)
		return
	}

	api.NoContent(c)
}

func ChangeUserPassword(c *gin.Context) {
	userID := c.Param("userID")

	var req models.ChangeUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.ChangeUserPassword(userID, req.Password); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrWeakPassword) {
			status = http.StatusBadRequest
		}
		api.Error(c, status, err)
		return
	}

	api.NoContent(c)
}

func DeleteUser(c *gin.Context) {
	userID := c.Param("userID")
	if err := services.DeleteUser(userID); err != nil {
		if errors.Is(err, services.ErrDeleteLastAdmin) {
			api.Error(c, http.StatusForbidden, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func ChangeUserRole(c *gin.Context) {
	userID := c.Param("userID")
	var req models.ChangeUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.ChangeUserRole(userID, req.Role); err != nil {
		if errors.Is(err, services.ErrDemoteLastAdmin) {
			api.Error(c, http.StatusForbidden, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := services.CreateUser(&req)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrWeakPassword), errors.Is(err, services.ErrUsernameAlreadyExists), errors.Is(err, services.ErrEmailAlreadyExists):
			status = http.StatusBadRequest
		}
		api.Error(c, status, err)
		return
	}

	api.Created(c, toUserResponse(user))
}

func ImportUsers(c *gin.Context) {
	// Get the file from the request
	file, err := c.FormFile("file")
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	defer src.Close()

	// Determine file type and parse
	var requests []models.CreateUserRequest

	fileType := c.PostForm("type") // json, csv, or excel
	if fileType == "" {
		// Auto-detect from extension
		ext := file.Filename
		if len(ext) > 4 {
			switch ext[len(ext)-4:] {
			case ".csv":
				fileType = "csv"
			case ".json":
				fileType = "json"
			case ".xlsx", ".xls":
				fileType = "excel"
			}
		}
	}

	switch fileType {
	case "json":
		requests, err = parseJSONUsers(src)
	case "csv":
		requests, err = parseCSVUsers(src)
	default:
		// Default to JSON
		requests, err = parseJSONUsers(src)
	}

	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	result, err := services.BatchImportUsers(requests)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, result)
}

// parseJSONUsers parses a JSON file containing an array of user data
func parseJSONUsers(src interface{ Read([]byte) (int, error) }) ([]models.CreateUserRequest, error) {
	// Read all content
	data := make([]byte, 0, 1024)
	buf := make([]byte, 1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	var requests []models.CreateUserRequest
	if err := json.Unmarshal(data, &requests); err != nil {
		return nil, err
	}

	return requests, nil
}

// parseCSVUsers parses a CSV file containing user data
// Expected headers: username,email,password,fullname,phone,role
func parseCSVUsers(src interface{ Read([]byte) (int, error) }) ([]models.CreateUserRequest, error) {
	// Read all content
	data := make([]byte, 0, 1024)
	buf := make([]byte, 1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	lines, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		return nil, err
	}

	if len(lines) < 2 {
		return nil, errors.New("CSV file must have a header row and at least one data row")
	}

	// Parse header to find column indices
	header := lines[0]
	colIndex := make(map[string]int)
	for i, h := range header {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// Check required columns
	if _, ok := colIndex["username"]; !ok {
		return nil, errors.New("CSV file must have a 'username' column")
	}
	if _, ok := colIndex["email"]; !ok {
		return nil, errors.New("CSV file must have an 'email' column")
	}
	if _, ok := colIndex["password"]; !ok {
		return nil, errors.New("CSV file must have a 'password' column")
	}

	var requests []models.CreateUserRequest
	for i := 1; i < len(lines); i++ {
		row := lines[i]
		if len(row) == 0 || (len(row) == 1 && row[0] == "") {
			continue
		}

		getVal := func(col string) string {
			idx, ok := colIndex[col]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}

		req := models.CreateUserRequest{
			Username: getVal("username"),
			Email:    getVal("email"),
			Password: getVal("password"),
			Fullname: getVal("fullname"),
			Phone:    getVal("phone"),
			Role:     getVal("role"),
		}

		// Default role to "user" if not specified
		if req.Role == "" {
			req.Role = app.UserRoleUser
		}

		requests = append(requests, req)
	}

	return requests, nil
}
