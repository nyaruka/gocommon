package cwatch

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// Some utility functions because the standard API is annoyingly verbose

func Dimension(name, value string) types.Dimension {
	return types.Dimension{Name: aws.String(name), Value: aws.String(value)}
}

func Datum(metric string, value float64, unit types.StandardUnit, dims ...types.Dimension) types.MetricDatum {
	return types.MetricDatum{
		MetricName: aws.String(metric),
		Dimensions: dims,
		Value:      aws.Float64(value),
		Unit:       unit,
	}
}
