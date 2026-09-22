// Package bill turns HTTP into calls on the Bill service.
package bill

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	billentity "github.com/nachoconques0/splitter_svc/backend/internal/entity/bill"
	splitterhttp "github.com/nachoconques0/splitter_svc/backend/internal/http"
	"github.com/nachoconques0/splitter_svc/backend/internal/model"
)

// Service is what this controller needs of the Bill service.
type Service interface {
	FindByID(ctx context.Context, id uuid.UUID) (billentity.Bill, error)
	ReplaceShareSet(ctx context.Context, id uuid.UUID, expectedVersion int64, shares billentity.ShareSet) (billentity.Bill, error)
}

// Controller serves the Bill routes.
type Controller struct {
	logger  *slog.Logger
	service Service
}

// New builds the controller over the given service.
func New(logger *slog.Logger, service Service) *Controller {
	return &Controller{logger: logger, service: service}
}

// ShareSet handles GET /bills/{id}/shares.
func (c *Controller) ShareSet(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		// Rejected here, so a malformed id never reaches the driver as a 500.
		splitterhttp.AbortWithError(ctx, http.StatusBadRequest, splitterhttp.ErrCodeInvalidBillID, "that is not a valid bill id")
		return
	}

	found, err := c.service.FindByID(ctx.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, billentity.ErrNotFound):
			splitterhttp.AbortWithError(ctx, http.StatusNotFound, splitterhttp.ErrCodeBillNotFound, "no bill with that id")
		default:
			splitterhttp.AbortWithInternalError(ctx, c.logger, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, shareSetResponse(found))
}

// shareSetResponse turns a Bill into the envelope both routes answer with.
func shareSetResponse(found billentity.Bill) model.ShareSetResponse {
	// An Amount is not stored: only the whole Share Set can be turned into
	// Amounts that sum to the Total.
	allocated := found.Allocate()

	shares := make([]model.Share, 0, len(allocated))
	for _, share := range allocated {
		shares = append(shares, model.Share{
			PersonID:   share.PersonID.String(),
			Name:       share.PersonName,
			Percentage: share.Percentage.Decimal(),
			Amount:     share.Amount.Decimal(),
		})
	}

	return model.ShareSetResponse{
		Bill: model.Bill{
			ID:          found.ID.String(),
			Description: found.Description,
			Total:       found.Total.Decimal(),
			Currency:    found.Total.Currency,
			Version:     found.Version,
		},
		Shares: shares,
	}
}
