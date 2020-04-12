package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/cccac/OpenWarn-Proxy/graph/generated"
	"github.com/cccac/OpenWarn-Proxy/graph/model"
)

func (r *queryResolver) ActiveAlerts(ctx context.Context, location *model.LocationInput) ([]model.Alert, error) {
	if location == nil {
		// No location filter provided, just return all active alerts
		var alerts []model.Alert
		for _, activeAlert := range r.Proxy.GetAllAlerts() {
			alert, err := model.FromAlert(activeAlert)
			if err != nil {
				return nil, err
			}
			alerts = append(alerts, alert)
		}

		return alerts, nil
	}

	var alerts []model.Alert
	for _, activeAlert := range r.Proxy.GetMatchingAlerts(model.ToLocation(location)) {
		alert, err := model.FromAlert(activeAlert)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
