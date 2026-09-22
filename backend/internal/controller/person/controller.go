// Package person turns HTTP into calls on the Person service.
package person

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	personentity "github.com/nachoconques0/splitter_svc/backend/internal/entity/person"
	splitterhttp "github.com/nachoconques0/splitter_svc/backend/internal/http"
	"github.com/nachoconques0/splitter_svc/backend/internal/model"
)

// Service is what this controller needs of the Person service.
type Service interface {
	Create(ctx context.Context, name string) (personentity.Person, error)
	List(ctx context.Context) ([]personentity.Person, error)
}

// Controller serves the Person routes.
type Controller struct {
	logger  *slog.Logger
	service Service
}

// New builds the controller over the given service.
func New(logger *slog.Logger, service Service) *Controller {
	return &Controller{logger: logger, service: service}
}

// Create handles POST /people.
func (c *Controller) Create(ctx *gin.Context) {
	var request model.CreatePersonRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		// No detail: a body that would not decode has no field to blame.
		splitterhttp.AbortWithError(ctx, http.StatusBadRequest, splitterhttp.ErrCodeInvalidRequestBody,
			"the request body is not a person")
		return
	}

	created, err := c.service.Create(ctx.Request.Context(), request.Name)
	if err != nil {
		switch {
		case errors.Is(err, personentity.ErrInvalidName):
			splitterhttp.AbortWithDetailedError(ctx, http.StatusUnprocessableEntity, splitterhttp.ErrCodeInvalidName,
				"a person needs a name that is not blank", "name")
		default:
			splitterhttp.AbortWithInternalError(ctx, c.logger, err)
		}
		return
	}

	ctx.JSON(http.StatusCreated, personResponse(created))
}

// List handles GET /people.
func (c *Controller) List(ctx *gin.Context) {
	people, err := c.service.List(ctx.Request.Context())
	if err != nil {
		splitterhttp.AbortWithInternalError(ctx, c.logger, err)
		return
	}

	// Length zero rather than nil, so nobody yet is [] and not null.
	response := model.PeopleResponse{People: make([]model.Person, 0, len(people))}
	for _, found := range people {
		response.People = append(response.People, personResponse(found))
	}

	ctx.JSON(http.StatusOK, response)
}

// personResponse turns a Person into the shape that crosses the wire.
func personResponse(found personentity.Person) model.Person {
	return model.Person{ID: found.ID.String(), Name: found.Name}
}
