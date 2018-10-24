package metrics

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	"github.com/openebs/scope-plugin5/k8s"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Query parameters for cortex agent.
const (
	iopsReadQuery        = "increase(openebs_reads[5m])/300"
	iopsWriteQuery       = "increase(openebs_writes[5m])/300"
	latencyReadQuery     = "((increase(openebs_read_time[5m]))/(increase(openebs_reads[5m])))/1000000"
	latencyWriteQuery    = "((increase(openebs_write_time[5m]))/(increase(openebs_writes[5m])))/1000000"
	throughputReadQuery  = "increase(openebs_read_block_count[5m])/(1024*1024*60*5)"
	throughputWriteQuery = "increase(openebs_write_block_count[5m])/(1024*1024*60*5)"
	// URL is the address of cortex agent.
	URL = "http://cortex-agent-service.maya-system.svc.cluster.local:80/api/v1/query?query="
)

// Map to store the query response.
var (
	readIops        = make(map[string]float64)
	writeIops       = make(map[string]float64)
	readLatency     = make(map[string]float64)
	writeLatency    = make(map[string]float64)
	readThroughput  = make(map[string]float64)
	writeThroughput = make(map[string]float64)
	querymap        = make(map[string]map[string]float64)
)

// Mutex is used to lock over metrics structure.
var Mutex = &sync.Mutex{}

// Response unmarshal the obtained Metric json
func Response(response []byte) (*Metrics, error) {
	result := new(Metrics)
	err := json.Unmarshal(response, &result)
	return result, err
}

//GetValues will get the values from the cortex agent.
func GetValues() map[string]PVMetrics {
	queries := []string{iopsReadQuery, iopsWriteQuery, latencyReadQuery, latencyWriteQuery, throughputReadQuery, throughputWriteQuery}
	for _, query := range queries {
		res, err := http.Get(URL + query)
		if err != nil {
			log.Error(err)
		}

		responseBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Error(err)
		}

		response, err := Response([]byte(responseBody))
		if err != nil {
			log.Error(err)
		}

		metrics := make(map[string]float64)

		for _, result := range response.Data.Result {
			if result.Value[1].(string) != "NaN" {
				floatVal, err := strconv.ParseFloat(result.Value[1].(string), 32)
				if err == nil {
					metrics[result.Metric.OpenebsPv] = floatVal
				} else {
					log.Error(err)
				}
			} else {
				metrics[result.Metric.OpenebsPv] = 0
			}
		}

		// switch query {
		// case iopsReadQuery:
		// 	readIops = metrics
		// case iopsWriteQuery:
		// 	writeIops = metrics
		// case latencyReadQuery:
		// 	readLatency = metrics
		// case latencyWriteQuery:
		// 	writeLatency = metrics
		// case throughputReadQuery:
		// 	readThroughput = metrics
		// case throughputWriteQuery:
		// 	writeThroughput = metrics
		// }
		// querymap := make(map[string]map[string]float64)
		querymap[query] = metrics
	}

	data := make(map[string]PVMetrics)
	if len(querymap[iopsReadQuery]) > 0 && len(querymap[iopsWriteQuery]) > 0 && len(querymap[latencyReadQuery]) > 0 && len(querymap[latencyWriteQuery]) > 0 && len(querymap[throughputReadQuery]) > 0 && len(querymap[throughputWriteQuery]) > 0 {
		// if len(readIops) > 0 && len(writeIops) > 0 && len(readLatency) > 0 && len(writeLatency) > 0 && len(readThroughput) > 0 && len(writeThroughput) > 0 {
		pvList, err := k8s.ClientSet.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
		if err != nil {
			log.Error(err)
		}

		pvUID := make(map[string]string)

		for _, p := range pvList.Items {
			pvUID[p.GetName()] = string(p.GetUID())
		}

		for pvName, iopsRead := range querymap[iopsReadQuery] {
			metrics := PVMetrics{
				ReadIops: iopsRead,
			}

			if val, ok := (querymap[iopsWriteQuery])[pvName]; ok {
				metrics.WriteIops = val
			}
			if val, ok := (querymap[latencyReadQuery])[pvName]; ok {
				metrics.ReadLatency = val
			}
			if val, ok := (querymap[latencyWriteQuery])[pvName]; ok {
				metrics.WriteLatency = val
			}
			if val, ok := (querymap[throughputReadQuery])[pvName]; ok {
				metrics.ReadThroughput = val
			}
			if val, ok := (querymap[throughputWriteQuery])[pvName]; ok {
				metrics.WriteThroughput = val
			}
			data[pvUID[pvName]] = metrics
		}
	}
	return data
}
