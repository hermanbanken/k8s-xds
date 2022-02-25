package internal

import (
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"

	"go.uber.org/zap"

	"github.com/google/uuid"
)

// GenerateSnapshot creates snapshot for each service
func GenerateSnapshot(services []string) (*cache.Snapshot, error) {
	// k8sEndPoints, err := getK8sEndPoints(services)
	// if err != nil {
	// 	zap.L().Error("Error while trying to get EndPoints from k8s cluster", zap.Error(err))
	// 	return nil, errors.New("Error while trying to get EndPoints from k8s cluster")
	// }

	// zap.L().Debug("K8s", zap.Any("EndPoints", k8sEndPoints))

	var eds []types.Resource
	var cds []types.Resource
	var rds []types.Resource
	var lds []types.Resource
	// for service, podEndPoints := range k8sEndPoints {
		// zap.L().Debug("Creating new XDS Entry", zap.String("service", service))
		// eds = append(eds, clusterLoadAssignment(podEndPoints, fmt.Sprintf("%s-cluster", service), "my-region", "my-zone")...)
		// cds = append(cds, createCluster(fmt.Sprintf("%s-cluster", service))...)
		// rds = append(rds, createRoute(fmt.Sprintf("%s-route", service), fmt.Sprintf("%s-vhost", service), fmt.Sprintf("%s-listener", service), fmt.Sprintf("%s-cluster", service))...)
		// lds = append(lds, createListener(fmt.Sprintf("%s-listener", service), fmt.Sprintf("%s-cluster", service), fmt.Sprintf("%s-route", service))...)
	// }

	version := uuid.New()
	zap.L().Debug("Creating Snapshot", zap.String("version", version.String()), zap.Any("EDS", eds), zap.Any("CDS", cds), zap.Any("RDS", rds), zap.Any("LDS", lds))
	snapshot := cache.NewSnapshot(version.String(), eds, cds, rds, lds, []types.Resource{}, []types.Resource{})

	if err := snapshot.Consistent(); err != nil {
		zap.L().Error("Snapshot inconsistency", zap.Any("snapshot", snapshot), zap.Error(err))
	}
	return &snapshot, nil
}