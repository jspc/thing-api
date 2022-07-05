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
	base      = "/api/thing"
)

// @Description Request body for a new 'thing`
type NewThing struct {
	// The name of the new thing
	Name string `json:"name" validate:"required,printascii,min=5" example:"my new thing"`
}

// @Description An object of the 'Thing' model
type Thing struct {
	// The id of this Thing
	ID string `json:"id"`

	// The name of this thing
	Name string `json:"name"`

	// The status of this thing, in enum ['creating', 'created', 'error']
	Status string `json:"status" enums:"creating,created,error"`

	// The time this thing was created at
	CreatedAt time.Time `json:"created_at"`

	// The time this thing was updated at
	UpdatedAt time.Time `json:"updated_at"`

	// Flooble holds the obviously important flooble value of this thing,
	// which is unique to this thing and should be cherished
	Flooble int `json:"flooble"`
}

type Error struct {
	M string `json:"msg"`
}

type API struct {
	r        *gin.Engine
	things   map[string]*Thing
	validate *validator.Validate
}

func New() (a API) {
	a.things = make(map[string]*Thing)
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
	docs.SwaggerInfo.Title = "The Amazing Thing API, with added Floobles"
	docs.SwaggerInfo.Description = "All the things, all the thing floobles, all the time"

	api := a.r.Group(base, a.validateToken)

	api.POST("/", a.NewThing)
	api.GET("/:id", a.LoadThing)
	api.DELETE("/:uuid", a.DeleteThing)

	return
}

func (a API) validateToken(g *gin.Context) {
	token := g.Request.Header.Get("Authorization")
	if token == "" || token != apiToken {
		g.AbortWithStatusJSON(400, a.errorMsg("missing or invalid authorization token"))

		return
	}

	g.Next()
}

// NewThing godoc
// @Summary create a new thing
// @Schemes
// @Description create a new thing
// @Tags New
// @Accept json
// @Produce json
// @Param        Authorization  header    string  true  "Authentication header"
// @Param thing body NewThing true "New thing"
// @Success 201 {object} Thing
// @Failure 400 {object} Error "The input object failed validation"
// @Failure 401 {object} Error "Missing or Invalid Authorization Token"
// @Failure 429 {string} string "Too many requests"
// @Router / [post]
func (a *API) NewThing(g *gin.Context) {
	u := uuid.Must(uuid.NewV4()).String()

	thing := new(NewThing)
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

	a.things[u] = &Thing{
		ID:        u,
		Name:      sanitiser.Sanitize(thing.Name),
		Status:    "creating",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Flooble:   rand.Int(),
	}

	g.JSON(http.StatusCreated, a.things[u])
}

// LoadThing godoc
// @Summary load a thing
// @Schemes
// @Description Load a thing from the thing API
// @Tags Get
// @Produce json
// @Param        Authorization  header    string  true  "Authentication header"
// @Param id path string true "Thing ID, see the ID field from a new Thing"
// @Success 200 {object} Thing
// @Failure 401 {object} Error "Missing or Invalid Authorization Token"
// @Failure 404 {object} Error "This thing does not exist"
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
// @Summary delete a thing
// @Schemes
// @Description Delete a thing from the thing API
// @Description Note: this endpoint will fail if the thing does not exist
// @Tags Delete
// @Produce plain
// @Param        Authorization  header    string  true  "Authentication header"
// @Param id path string true "Thing ID, see the ID field from a new Thing"
// @Success 200 {string} string "The thing was successfully deleted"
// @Failure 401 {object} Error "Missing or Invalid Authorization Token"
// @Failure 404 {object} Error "This thing does not exist"
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
