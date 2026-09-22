package bill

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	billentity "github.com/nachoconques0/splitter_svc/backend/internal/entity/bill"
	splitterhttp "github.com/nachoconques0/splitter_svc/backend/internal/http"
	"github.com/nachoconques0/splitter_svc/backend/internal/model"
)

// ReplaceShareSet handles PUT /bills/{id}/shares.
func (c *Controller) ReplaceShareSet(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		splitterhttp.AbortWithError(ctx, http.StatusBadRequest, splitterhttp.ErrCodeInvalidBillID, "that is not a valid bill id")
		return
	}

	var request model.ReplaceShareSetRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		abortMalformed(ctx, "the request body is not a share set", "")
		return
	}

	// Versions start at 1, so zero means it was never sent.
	if request.Version == 0 {
		abortMalformed(ctx, "a save has to carry the version the bill was read at", "version")
		return
	}

	shares, err := toShareSet(request)
	if err != nil {
		// Unreadable is 400; read and refused is 422, below.
		abortMalformed(ctx, "a value in the request body is not what it has to be", err.Error())
		return
	}

	replaced, err := c.service.ReplaceShareSet(ctx.Request.Context(), id, request.Version, shares)
	if err != nil {
		c.abortReplaceFailure(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, shareSetResponse(replaced))
}

// toShareSet parses the request into the types the domain works in.
func toShareSet(request model.ReplaceShareSetRequest) (billentity.ShareSet, error) {
	shares := make(billentity.ShareSet, 0, len(request.Shares))

	for i, input := range request.Shares {
		personID, err := uuid.Parse(input.PersonID)
		if err != nil {
			return nil, fmt.Errorf("shares[%d].person_id is not a valid person id", i)
		}

		percentage, err := billentity.ParsePercentage(input.Percentage.String())
		if errors.Is(err, billentity.ErrMalformedPercentage) {
			return nil, fmt.Errorf("shares[%d].percentage is not a percentage carrying two decimal places", i)
		}
		if err != nil {
			return nil, fmt.Errorf("shares[%d].percentage could not be read: %w", i, err)
		}

		shares = append(shares, billentity.Share{PersonID: personID, Percentage: percentage})
	}

	return shares, nil
}

// abortMalformed refuses a request that could not be read as a save.
func abortMalformed(ctx *gin.Context, message, detail string) {
	splitterhttp.AbortWithDetailedError(ctx, http.StatusBadRequest, splitterhttp.ErrCodeInvalidRequestBody,
		message, detail)
}

// Anything unnamed becomes a 500, so a forgotten rule cannot leak a raw error.
func (c *Controller) abortReplaceFailure(ctx *gin.Context, err error) {
	var (
		sum        billentity.SumError
		duplicate  billentity.DuplicatePersonError
		unknown    billentity.UnknownPersonError
		outOfRange billentity.PercentageOutOfRangeError
	)

	switch {
	case errors.Is(err, billentity.ErrNotFound):
		splitterhttp.AbortWithError(ctx, http.StatusNotFound, splitterhttp.ErrCodeBillNotFound, "no bill with that id")

	case errors.Is(err, billentity.ErrModified):
		splitterhttp.AbortWithError(ctx, http.StatusConflict, splitterhttp.ErrCodeBillModified,
			"this bill changed since you loaded it, so saving would overwrite that change")

	case errors.As(err, &sum):
		splitterhttp.AbortWithDetailedError(ctx, http.StatusUnprocessableEntity, splitterhttp.ErrCodeSharesMustSumTo100,
			"the percentages add up to "+sum.Sum.Decimal()+", but a share set must total exactly "+billentity.Whole.Decimal(),
			sum.Sum.Decimal())

	case errors.As(err, &duplicate):
		splitterhttp.AbortWithDetailedError(ctx, http.StatusUnprocessableEntity, splitterhttp.ErrCodeDuplicatePerson,
			"the same person appears twice on this bill", duplicate.PersonID.String())

	case errors.As(err, &unknown):
		splitterhttp.AbortWithDetailedError(ctx, http.StatusUnprocessableEntity, splitterhttp.ErrCodeUnknownPerson,
			"that share set names someone who does not exist", strings.Join(idsOf(unknown.PersonIDs), ","))

	case errors.As(err, &outOfRange):
		splitterhttp.AbortWithDetailedError(ctx, http.StatusUnprocessableEntity, splitterhttp.ErrCodeInvalidPercentage,
			"a percentage of "+outOfRange.Percentage.Decimal()+" is not one a share can claim",
			outOfRange.Percentage.Decimal())

	default:
		splitterhttp.AbortWithInternalError(ctx, c.logger, err)
	}
}

// idsOf renders person ids for the error envelope's detail.
func idsOf(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}
