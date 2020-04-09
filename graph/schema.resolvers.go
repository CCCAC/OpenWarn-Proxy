package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/cccac/OpenWarn-Proxy/graph/generated"
	"github.com/cccac/OpenWarn-Proxy/graph/model"
	"github.com/cccac/OpenWarn-Proxy/proxy"
)

func (r *queryResolver) ActiveAlerts(ctx context.Context, location *model.LocationInput) ([]*model.Alert, error) {
	if location == nil {
		var alerts []*model.Alert
		for _, alert := range r.Proxy.GetAllAlerts() {
			alerts = append(alerts, &model.Alert{
				ID:      string(alert.Identifier),
				Message: string(alert.Info[0].Description),
			})
		}
		return alerts, nil
	}

	// Only alerts for a specific location
	c := proxy.Coordinate{
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
	}

	var alerts []*model.Alert
	proxyAlerts := r.Proxy.GetMatchingAlerts(c)
	for _, proxyAlert := range proxyAlerts {
		alert := model.Alert{
			ID:      string(proxyAlert.Identifier),
			Message: proxyAlert.Info[0].Description,
		}
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
