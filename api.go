package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"
	"github.com/jspc/thing-api/docs"
	"github.com/julianshen/gin-limiter"
	"github.com/microcosm-cc/bluemonday"
	files "github.com/swaggo/files"
	swag "github.com/swaggo/gin-swagger"
)

// generated via: `echo devex_$(dd if=/dev/random bs=1 count=64 | base64 -w 0 | tr -cd '[:alnum:]._-')`
var (
	apiToken  = "devex_JIJrVvoMj3gy2xHTEo0QoGMpepCl0rJ8d1pkN3VgjjbMzJzerjnzLUhoNKMRoT8aTqUjoTaZxgT9fXReN2aw"
	sanitiser = bluemonday.StrictPolicy()
	base      = "/api/tokens"

	description = `The DevEx auth API is used to generate API Tokens for
various DevEx services and integrations.`
)

// @Description Request body for a new Token
type NewToken struct {
	// The name of the new thing
	Name string `json:"name" validate:"required,printascii,min=5,max=64,excludes=;" example:"my new thing"`
}

// @Description A user created Token, including the status and name
// @Description of the Token
type Token struct {
	// The id of this Token
	ID string `json:"id"`

	// The name of this Token, as specified by the user
	Name string `json:"name"`

	// The status of this Token, in enum ['creating', 'created', 'error']
	Status string `json:"status" enums:"creating,created,error"`

	// The time this token was created at
	CreatedAt time.Time `json:"created_at"`

	// The time this token was updated at
	UpdatedAt time.Time `json:"updated_at"`

	// Value represents the user token associated with this object and
	// is used when interacting with other APIs.
	//
	// The token Value should be stored securely, given the amount
	// of power it has
	Value int `json:"value"`
}

// @Description Error is a generic model for surfacing errors to users
type Error struct {
	M string `json:"msg"`
}

type API struct {
	r        *gin.Engine
	things   map[string]*Token
	validate *validator.Validate
}

func New() (a API) {
	a.things = make(map[string]*Token)
	a.validate = validator.New()

	a.r = gin.New()
	a.r.GET("/swagger/*any", swag.WrapHandler(files.Handler))

	a.r.Use(gin.Logger())
	a.r.Use(gin.Recovery())
	a.r.Use(cors.Default())
	a.r.Use(gzip.Gzip(gzip.DefaultCompression))
	a.r.Use(requestid.New())

	lm := ginlimiter.NewRateLimiter(time.Second, 1, func(ctx *gin.Context) (string, error) {
		return ctx.ClientIP(), nil
	})

	a.r.Use(lm.Middleware())

	docs.SwaggerInfo.BasePath = base
	docs.SwaggerInfo.Title = "DevEx Auth API"
	docs.SwaggerInfo.Description = description

	api := a.r.Group(base, a.validateAuthToken)

	api.GET("/", a.Things)
	api.POST("/", a.NewThing)
	api.GET("/:id", a.LoadThing)
	api.DELETE("/:uuid", a.DeleteThing)

	return
}

func (a API) validateAuthToken(g *gin.Context) {
	token := g.Request.Header.Get("Authorization")
	if token == "" || token != apiToken {
		g.AbortWithStatusJSON(400, a.errorMsg("missing or invalid authorization token"))

		return
	}

	g.Next()
}

// Things godoc
// @Summary return a list of a tokens owned by the current user
// @Schemes
// @Description Return a list of all DevEx tokens owned by the
// @Description currently authenticated user
// @Tags All
// @Accept json
// @Produce json
// @Param        Authorization  header    string  true  "Authentication header"
// @Success 200 {array} Token
// @Failure 401 {object} Error "Missing or Invalid Authorization header"
// @Failure 429 {string} string "Too many requests"
// @Router / [get]
func (a *API) Things(g *gin.Context) {
	things := make([]Token, 0, len(a.things))

	for _, t := range a.things {
		things = append(things, *t)
	}

	g.JSON(http.StatusOK, things)
}

// NewThing godoc
// @Summary create a new token
// @Schemes
// @Description Accept a name and generate a new Thing, assigned
// @Description to the currently authenticated user.
// @Description A successful call to this end point will return status 201.
// @Description You must make further GETs on the returned resource in order
// @Description to determine whether the resource has been created successfully.
// @Tags New
// @Accept json
// @Produce json
// @Param        Authorization  header    string  true  "Authentication header"
// @Param thing body NewToken true "Body containing the name of the new token to create"
// @Success 201 {object} Token
// @Failure 400 {object} Error "The input object failed validation"
// @Failure 401 {object} Error "Missing or Invalid Authorization Header"
// @Failure 429 {string} string "Too many requests"
// @Router / [post]
func (a *API) NewThing(g *gin.Context) {
	u := uuid.Must(uuid.NewV4()).String()

	thing := new(NewToken)
	err := g.BindJSON(thing)
	if err != nil {
		g.AbortWithStatusJSON(400, a.errorMsg("invalid input"))

		log.Print(err)

		return
	}
	g.Abort()

	err = a.validate.Struct(thing)
	if err != nil {
		g.AbortWithStatusJSON(400, a.errorMsg("invalid input"))

		log.Print(err)

		return
	}

	a.things[u] = &Token{
		ID:        u,
		Name:      sanitiser.Sanitize(thing.Name),
		Status:    "creating",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Value:     rand.Int(),
	}

	g.JSON(http.StatusCreated, a.things[u])
}

// LoadThing godoc
// @Summary load a token
// @Schemes
// @Description Load a specific token, by ID, for the currently
// @Description authenticated user, returning a 404 when no such token
// @Description can be found
// @Tags Get
// @Produce json
// @Param        Authorization  header    string  true  "Authentication header"
// @Param id path string true "Token ID"
// @Success 200 {object} Token
// @Failure 401 {object} Error "Missing or Invalid Authorization Header"
// @Failure 404 {object} Error "This Token does not exist"
// @Failure 429 {string} string "Too many requests"
// @Router /{id} [get]
func (a *API) LoadThing(g *gin.Context) {
	id := g.Param("id")

	t, ok := a.things[id]
	if !ok {
		g.AbortWithStatusJSON(http.StatusNotFound, a.errorMsg("not found"))

		return
	}

	if t.Status != "creating" {
		g.JSON(http.StatusOK, t)

		return
	}

	now := time.Now()
	if now.Sub(t.CreatedAt) > time.Second*3 {
		switch rand.Intn(10) {
		case 0:
			t.Status = "error"
			t.UpdatedAt = time.Now()

		case 1, 2, 3:
			t.Status = "created"
			t.UpdatedAt = time.Now()
		}
	}

	a.things[id] = t

	g.JSON(http.StatusOK, t)
}

// LoadThing godoc
// @Summary delete a token
// @Schemes
// @Description Delete the specified Token,
// @Description returning a 404 if the Token ID is unrecognised
// @Tags Delete
// @Produce plain
// @Param        Authorization  header    string  true  "Authentication header"
// @Param id path string true "Token ID"
// @Success 200 {string} string "The token was successfully deleted"
// @Failure 401 {object} Error "Missing or Invalid Authorization Header"
// @Failure 404 {object} Error "This token does not exist"
// @Failure 429 {string} string "Too many requests"
// @Router /{id} [delete]
func (a *API) DeleteThing(g *gin.Context) {
	id := g.Param("id")

	t, ok := a.things[id]
	if !ok {
		g.AbortWithStatusJSON(http.StatusNotFound, a.errorMsg("not found"))

		return
	}

	if t.Status == "creating" {
		g.AbortWithStatusJSON(http.StatusBadRequest, a.errorMsg("cannot delete things in creating state"))

		return
	}

	delete(a.things, id)

	g.String(http.StatusOK, "deleted")
}

func (a API) errorMsg(s string) Error {
	return Error{s}
}
