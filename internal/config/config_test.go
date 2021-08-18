package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfigFile(t *testing.T) {

	result, err := ReadConfigFile("../../configuration/test/test_1.yml")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(result.Gauge))

	gaugeQuery0 := result.Gauge[0]
	assert.Equal(t, "abc", gaugeQuery0.Query)
	assert.Equal(t, "/queries/metric1_name.sql", gaugeQuery0.File)

	gaugeQuery1 := result.Gauge[1]
	assert.Equal(t, "def", gaugeQuery1.Query)
	assert.Equal(t, "/queries/metric2_name.sql", gaugeQuery1.File)

	assert.Equal(t, 3, len(result.Counter))

	counterQuery0 := result.Counter[0]
	assert.Equal(t, "abc_2", counterQuery0.Query)
	assert.Equal(t, "/queries/metric3_name.sql", counterQuery0.File)

	counterQuery1 := result.Counter[1]
	assert.Equal(t, "def_2", counterQuery1.Query)
	assert.Equal(t, "/queries/metric4_name.sql", counterQuery1.File)

	counterQuery2 := result.Counter[2]
	assert.Equal(t, "ghi_2", counterQuery2.Query)
	assert.Equal(t, "/queries/metric5_name.sql", counterQuery2.File)
}
