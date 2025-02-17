// Copyright 2023 Antrea Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// Use a long timeout as it takes ~500s to complete a single Spark job on
	// Kind testbed
	tadjobCompleteTimeout = 10 * time.Minute
	tadstartCmd           = "./theia throughput-anomaly-detection run"
	tadstatusCmd          = "./theia throughput-anomaly-detection status"
	tadlistCmd            = "./theia throughput-anomaly-detection list"
	taddeleteCmd          = "./theia throughput-anomaly-detection delete"
	tadretrieveCmd        = "./theia throughput-anomaly-detection retrieve"
)

var e2eMutex sync.Mutex

func TestAnomalyDetection(t *testing.T) {
	config := FlowVisibilitySetUpConfig{
		withSparkOperator:     true,
		withGrafana:           false,
		withClickHouseLocalPv: false,
		withFlowAggregator:    true,
	}
	data, _, _, err := setupTestForFlowVisibility(t, config)
	if err != nil {
		t.Fatalf("Error when setting up test: %v", err)
	}
	defer func() {
		teardownTest(t, data)
		teardownFlowVisibility(t, data, config)
	}()

	clientset := data.clientset
	kubeconfig, err := data.provider.GetKubeconfigPath()
	require.NoError(t, err)
	connect, pf, err := SetupClickHouseConnection(clientset, kubeconfig)
	require.NoError(t, err)
	if pf != nil {
		defer pf.Stop()
	}

	t.Run("testThroughputAnomalyDetectionAlgo", func(t *testing.T) {
		testThroughputAnomalyDetectionAlgo(t, data, connect)
	})

	t.Run("testAnomalyDetectionStatus", func(t *testing.T) {
		testAnomalyDetectionStatus(t, data, connect)
	})

	t.Run("testAnomalyDetectionList", func(t *testing.T) {
		testAnomalyDetectionList(t, data, connect)
	})

	t.Run("TestAnomalyDetectionDelete", func(t *testing.T) {
		testAnomalyDetectionDelete(t, data, connect)
	})

	t.Run("TestAnomalyDetectionRetrieve", func(t *testing.T) {
		testAnomalyDetectionRetrieve(t, data, connect)
	})

	t.Run("testTADCleanAfterTheiaMgrResync", func(t *testing.T) {
		testTADCleanAfterTheiaMgrResync(t, data)
	})
}

func prepareFlowTable(t *testing.T, connect *sql.DB) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		populateFlowTable(t, connect)
	}()
	wg.Wait()
}

// Example output: Successfully created Throughput Anomaly Detection job with name tad-eec9d1be-7204-4d50-8f57-d9c8757a2668
func testThroughputAnomalyDetectionAlgo(t *testing.T, data *TestData, connect *sql.DB) {
	prepareFlowTable(t, connect)
	stdout, jobName, err := tadrunJob(t, data, "ARIMA", "None")
	require.NoError(t, err)
	assert := assert.New(t)
	assert.Containsf(stdout, fmt.Sprintf("Successfully started Throughput Anomaly Detection job with name: %s", jobName), "stdout: %s", stdout)
	err = data.podWaitForReady(defaultTimeout, jobName+"-driver", flowVisibilityNamespace)
	require.NoError(t, err)
	_, err = taddeleteJob(t, data, jobName)
	require.NoError(t, err)
}

// Example output: Status of this anomaly detection job is COMPLETED
func testAnomalyDetectionStatus(t *testing.T, data *TestData, connect *sql.DB) {
	prepareFlowTable(t, connect)
	_, jobName, err := tadrunJob(t, data, "ARIMA", "None")
	require.NoError(t, err)
	stdout, err := tadgetJobStatus(t, data, jobName)
	require.NoError(t, err)
	assert := assert.New(t)
	assert.Containsf(stdout, "Status of this anomaly detection job is", "stdout: %s", stdout)
	err = data.podWaitForReady(defaultTimeout, jobName+"-driver", flowVisibilityNamespace)
	require.NoError(t, err)
	_, err = taddeleteJob(t, data, jobName)
	require.NoError(t, err)
}

// Example output:
// CreationTime          CompletionTime        Name                                    Status
// 2022-06-17 15:03:24   N/A                   tad-615026a0-1856-4107-87d9-08f7d69819ae RUNNING
// 2022-06-17 15:03:22   2022-06-17 18:08:37   tad-7bebe4f9-408b-4dd8-9d63-9dc538073089 COMPLETED
// 2022-06-17 15:03:39   N/A                   tad-c7a9e768-559a-4bfb-b0c8-a0291b4c208c SUBMITTED
func testAnomalyDetectionList(t *testing.T, data *TestData, connect *sql.DB) {
	prepareFlowTable(t, connect)
	_, jobName, err := tadrunJob(t, data, "ARIMA", "None")
	require.NoError(t, err)
	stdout, err := tadlistJobs(t, data)
	require.NoError(t, err)
	assert := assert.New(t)
	assert.Containsf(stdout, "CreationTime", "stdout: %s", stdout)
	assert.Containsf(stdout, "CompletionTime", "stdout: %s", stdout)
	assert.Containsf(stdout, "Name", "stdout: %s", stdout)
	assert.Containsf(stdout, "Status", "stdout: %s", stdout)
	assert.Containsf(stdout, jobName, "stdout: %s", stdout)
	err = data.podWaitForReady(defaultTimeout, jobName+"-driver", flowVisibilityNamespace)
	require.NoError(t, err)
	_, err = taddeleteJob(t, data, jobName)
	require.NoError(t, err)
}

// Example output: Successfully deleted anomaly detection job with name tad-eec9d1be-7204-4d50-8f57-d9c8757a2668
func testAnomalyDetectionDelete(t *testing.T, data *TestData, connect *sql.DB) {
	prepareFlowTable(t, connect)
	_, jobName, err := tadrunJob(t, data, "ARIMA", "None")
	require.NoError(t, err)
	err = data.podWaitForReady(defaultTimeout, jobName+"-driver", flowVisibilityNamespace)
	require.NoError(t, err)
	stdout, err := taddeleteJob(t, data, jobName)
	require.NoError(t, err)
	assert := assert.New(t)
	assert.Containsf(stdout, "Successfully deleted anomaly detection job with name", "stdout: %s", stdout)
	stdout, err = tadlistJobs(t, data)
	require.NoError(t, err)
	assert.NotContainsf(stdout, jobName, "Still found deleted job in list command stdout: %s", stdout)
}

// Example Output
// id                                   destinationServicePortName flowEndSeconds       throughput       aggType        algoType       algoCalc               anomaly
// 5ca4413d-6730-463e-8f95-86032ba28a4f test_serviceportname       2022-08-11T08:24:54Z 5.0024845485e+10 svc            ARIMA          2.0863933021708477e+10 true
// 5ca4413d-6730-463e-8f95-86032ba28a4f test_serviceportname       2022-08-11T08:34:54Z 2.5003930638e+11 svc            ARIMA          1.9138281301304165e+10 true
func testAnomalyDetectionRetrieve(t *testing.T, data *TestData, connect *sql.DB) {
	algoNames := []string{"ARIMA", "EWMA", "DBSCAN"}
	// agg_type 'podLabel' stands for agg_type 'pod' with pod-label argument
	// agg_type 'podName' stands for agg_type 'pod' with pod-name argument
	agg_types := []string{"None", "podName", "podLabel", "svc", "external"}
	// Select random algo for the agg_types
	aggTypeToAlgoNameMap := make(map[string]string)
	for _, agg_type := range agg_types {
		aggTypeToAlgoNameMap[agg_type] = algoNames[randInt(t, int64(len(algoNames)))]
	}
	// Create a worker pool with maximum size of 3
	pool := make(chan int, len(algoNames))
	var (
		wg      sync.WaitGroup
		poolIdx int
	)
	prepareFlowTable(t, connect)
	result_map := map[string]map[string]string{
		"ARIMA": {
			"4.005": "true",
			"1.000": "true",
			"5.000": "true",
			"2.500": "true",
			"5.002": "true",
			"2.003": "true",
			"2.002": "true",
		},
		"EWMA": {
			"4.004": "true",
			"4.005": "true",
			"4.006": "true",
			"5.000": "true",
			"2.002": "true",
			"2.003": "true",
			"2.500": "true",
		},
		"DBSCAN": {
			"1.000": "true",
			"1.005": "true",
			"5.000": "true",
			"3.260": "true",
			"2.058": "true",
			"5.002": "true",
			"5.027": "true",
			"2.500": "true",
			"1.029": "true",
			"1.630": "true"},
	}
	assert_variable_map := map[string]map[string]int{
		"None": {
			"tadoutputArray_len": 12,
			"anomaly_output_idx": 11,
			"throughput_idx":     7},
		"podName": {
			"tadoutputArray_len": 10,
			"anomaly_output_idx": 9,
			"throughput_idx":     5},
		"podLabel": {
			"tadoutputArray_len": 9,
			"anomaly_output_idx": 8,
			"throughput_idx":     4},
		"external": {
			"tadoutputArray_len": 8,
			"anomaly_output_idx": 7,
			"throughput_idx":     3},
		"svc": {
			"tadoutputArray_len": 8,
			"anomaly_output_idx": 7,
			"throughput_idx":     3},
	}
	poolIdx = 0
	for agg_type, algoName := range aggTypeToAlgoNameMap {
		poolIdx += 1
		if agg_type == "None" {
			for _, algo := range algoNames {
				algoName = algo
				wg.Add(1)
				pool <- poolIdx
				go executeRetrieveTest(t, data, algoName, agg_type, result_map, assert_variable_map, pool, &wg)
			}
		} else {
			wg.Add(1)
			pool <- poolIdx
			go executeRetrieveTest(t, data, algoName, agg_type, result_map, assert_variable_map, pool, &wg)
		}
	}
	wg.Wait()
}

func executeRetrieveTest(t *testing.T, data *TestData, algo, agg_type string, result_map map[string]map[string]string, assert_variable_map map[string]map[string]int, pool chan int, wg *sync.WaitGroup) {
	var stdout string
	defer func() {
		<-pool
		wg.Done()
	}()
	_, jobName, err := tadrunJob(t, data, algo, agg_type)
	require.NoError(t, err)
	err = data.podWaitForReady(defaultTimeout, jobName+"-driver", flowVisibilityNamespace)
	require.NoError(t, err)
	err = waitTADJobComplete(t, data, jobName, tadjobCompleteTimeout)
	require.NoError(t, err)
	stdout, err = tadretrieveJobResult(t, data, jobName)
	require.NoError(t, err)
	resultArray := strings.Split(stdout, "\n")
	assert := assert.New(t)
	length := len(resultArray)
	assert.GreaterOrEqualf(length, 3, "stdout: %s", stdout)
	assert.Containsf(stdout, "throughput", "stdout: %s", stdout)
	assert.Containsf(stdout, "algoCalc", "stdout: %s", stdout)
	assert.Containsf(stdout, "anomaly", "stdout: %s", stdout)
	for i := 1; i < length; i++ {
		// check metrics' value
		resultArray[i] = strings.TrimSpace(resultArray[i])
		if resultArray[i] != "" {
			resultArray[i] = strings.ReplaceAll(resultArray[i], "\t", " ")
			tadoutputArray := strings.Fields(resultArray[i])
			anomaly_output := tadoutputArray[assert_variable_map[agg_type]["anomaly_output_idx"]]
			throughput := tadoutputArray[assert_variable_map[agg_type]["throughput_idx"]][:5]
			assert.Equal(assert_variable_map[agg_type]["tadoutputArray_len"], len(tadoutputArray), "tadoutputArray: %s", tadoutputArray)
			switch algo {
			case "ARIMA":
				assert.Equal(result_map["ARIMA"][throughput], anomaly_output, "Anomaly outputs dont match in tadoutputArray: %s", tadoutputArray)
			case "EWMA":
				assert.Equal(result_map["EWMA"][throughput], anomaly_output, "Anomaly outputs dont match in tadoutputArray: %s", tadoutputArray)
			case "DBSCAN":
				assert.Equal(result_map["DBSCAN"][throughput], anomaly_output, "Anomaly outputs dont match in tadoutputArray: %s", tadoutputArray)
			}
		}
	}
	_, err = taddeleteJob(t, data, jobName)
	require.NoError(t, err)
}

// waitJobComplete waits for the anomaly detection Spark job completes
func waitTADJobComplete(t *testing.T, data *TestData, jobName string, timeout time.Duration) error {
	e2eMutex.Lock()
	defer e2eMutex.Unlock()
	stdout := ""
	err := wait.PollImmediate(defaultInterval, timeout, func() (bool, error) {
		stdout, err := tadgetJobStatus(t, data, jobName)
		if err != nil {
			if strings.Contains(err.Error(), "TLS handshake timeout") {
				return false, nil
			}
		} else {
			require.NoError(t, err)
		}
		if strings.Contains(stdout, "Status of this anomaly detection job is COMPLETED") {
			return true, nil
		}
		// Keep trying
		return false, nil
	})
	if err == wait.ErrWaitTimeout {
		return fmt.Errorf("anomaly detection Spark job not completed after %v\nstatus:%s", timeout, stdout)
	} else if err != nil {
		return err
	}
	return nil
}

func tadrunJob(t *testing.T, data *TestData, algotype, agg_type string) (stdout string, jobName string, err error) {
	e2eMutex.Lock()
	defer e2eMutex.Unlock()
	var agg_flow_ext, ext string
	newjobcmd := tadstartCmd + " --algo " + algotype + " --driver-memory 1G --start-time 2022-08-11T06:26:50 --end-time 2022-08-12T08:26:54"
	switch agg_type {
	case "podName":
		agg_flow_ext = " --agg-flow pod"
		ext = " --pod-name test_podName"
	case "podLabel":
		agg_flow_ext = " --agg-flow pod"
		ext = " --pod-label test_key:test_value"
	case "external":
		agg_flow_ext = fmt.Sprintf(" --agg-flow %s", agg_type)
		ext = " --external-ip 10.10.1.33"
	case "svc":
		agg_flow_ext = fmt.Sprintf(" --agg-flow %s", agg_type)
		ext = " --svc-port-name test_serviceportname"
	}
	newjobcmd = newjobcmd + agg_flow_ext + ext
	stdout, jobName, err = RunJob(t, data, newjobcmd)
	if err != nil {
		return "", "", err
	}
	return stdout, jobName, nil
}

func tadgetJobStatus(t *testing.T, data *TestData, jobName string) (stdout string, err error) {
	cmd := fmt.Sprintf("%s %s", tadstatusCmd, jobName)
	stdout, err = GetJobStatus(t, data, cmd)
	if err != nil {
		return "", err
	}
	return stdout, nil
}

func tadlistJobs(t *testing.T, data *TestData) (stdout string, err error) {
	stdout, err = ListJobs(t, data, tadlistCmd)
	if err != nil {
		return "", err
	}
	return stdout, nil
}

func taddeleteJob(t *testing.T, data *TestData, jobName string) (stdout string, err error) {
	e2eMutex.Lock()
	defer e2eMutex.Unlock()
	cmd := fmt.Sprintf("%s %s", taddeleteCmd, jobName)
	stdout, err = DeleteJob(t, data, cmd)
	if err != nil {
		return "", err
	}
	return stdout, nil
}

func tadretrieveJobResult(t *testing.T, data *TestData, jobName string) (stdout string, err error) {
	e2eMutex.Lock()
	defer e2eMutex.Unlock()
	cmd := fmt.Sprintf("%s %s", tadretrieveCmd, jobName)
	stdout, err = RetrieveJobResult(t, data, cmd)
	if err != nil {
		return "", err
	}
	return stdout, nil
}

func addFakeRecordforTAD(t *testing.T, stmt *sql.Stmt) {
	flowStartSeconds, _ := time.Parse("2006-01-02T15:04:05", "2022-08-11T06:26:54")
	flowEndSeconds, _ := time.Parse("2006-01-02T15:04:05", "2022-08-11T07:26:54")
	sourceIP := "10.10.1.25"
	sourceTransportPort := 58076
	destinationIP := "10.10.1.33"
	destinationTransportPort := 5201
	protocolIndentifier := 6
	sourcePodNamespace := "test_namespace"
	sourcePodName := "test_podName"
	destinationPodName := "test_podName"
	destinationPodNamespace := "test_namespace"
	sourcePodLabels := "{test_key:test_value}"
	destinationPodLabels := "{test_key:test_value}"
	destinationServicePortName := "test_serviceportname"
	flowtype := 3

	throughputs := []int64{
		4007380032, 4006917952, 4004471308, 4005277827, 4005486294,
		4005435632, 4006917952, 4004471308, 4005277827, 4005486294,
		4005435632, 4006917952, 4004471308, 4005277827, 4005486294,
		4005435632, 4006917952, 4004471308, 4005277827, 4005486294,
		4005435632, 4006917952, 4004471308, 4005277827, 4005486294,
		4005435632, 4006917952, 4004471308, 4005277827, 4005486294,
		4005435632, 4006917952, 4004471308, 4005277827, 4005486294,
		4005435632, 4004465468, 4005336400, 4006201196, 4005546675,
		4005703059, 4004631769, 4006915708, 4004834307, 4005943619,
		4005760579, 4006503308, 4006580124, 4006524102, 4005521494,
		4004706899, 4006355667, 4006373555, 4005542681, 4006120227,
		4003599734, 4005561673, 4005682768, 10004969097, 4005517222,
		1005533779, 4005370905, 4005589772, 4005328806, 4004926121,
		4004496934, 4005615814, 4005798822, 50007861276, 4005396697,
		4005148294, 4006448435, 4005355097, 4004335558, 4005389043,
		4004839744, 4005556492, 4005796992, 4004497248, 4005988134,
		205881027, 4004638304, 4006191046, 4004723289, 4006172825,
		4005561235, 4005658636, 4006005936, 3260272025, 4005589772}
	for idx, throughput := range throughputs {
		_, err := stmt.Exec(
			flowStartSeconds,
			flowEndSeconds.Add(time.Minute*time.Duration(idx)),
			time.Now(),
			time.Now(),
			0,
			sourceIP,
			destinationIP,
			sourceTransportPort,
			destinationTransportPort,
			protocolIndentifier,
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			sourcePodName,
			sourcePodNamespace,
			fmt.Sprintf("NodeName-%d", randInt(t, MaxInt32)),
			destinationPodName,
			destinationPodNamespace,
			fmt.Sprintf("NodeName-%d", randInt(t, MaxInt32)),
			getRandIP(t),
			uint16(randInt(t, 65535)),
			destinationServicePortName,
			fmt.Sprintf("PolicyName-%d", randInt(t, MaxInt32)),
			fmt.Sprintf("PolicyNameSpace-%d", randInt(t, MaxInt32)),
			fmt.Sprintf("PolicyRuleName-%d", randInt(t, MaxInt32)),
			1,
			1,
			fmt.Sprintf("PolicyName-%d", randInt(t, MaxInt32)),
			fmt.Sprintf("PolicyNameSpace-%d", randInt(t, MaxInt32)),
			fmt.Sprintf("PolicyRuleName-%d", randInt(t, MaxInt32)),
			1,
			1,
			"tcpState",
			flowtype,
			sourcePodLabels,
			destinationPodLabels,
			uint64(throughput),
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			uint64(randInt(t, MaxInt32)),
			"",
			"",
			"",
		)
		require.NoError(t, err)
	}
}

func writeTADRecords(t *testing.T, connect *sql.DB, wg *sync.WaitGroup) {
	defer wg.Done()
	err := wait.PollImmediate(5*defaultInterval, defaultTimeout, func() (bool, error) {
		// Test ping DB
		err := connect.Ping()
		if err != nil {
			return false, nil
		}
		// Test open Transaction
		tx, err := connect.Begin()
		if err != nil {
			return false, nil
		}
		stmt, _ := tx.Prepare(insertQueryflowtable)
		defer stmt.Close()
		addFakeRecordforTAD(t, stmt)

		if err != nil {
			return false, nil
		}
		err = tx.Commit()
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	assert.NoError(t, err, "Unable to commit successfully to ClickHouse")
}

func populateFlowTable(t *testing.T, connect *sql.DB) {
	var wg sync.WaitGroup
	wg.Add(1)
	go writeTADRecords(t, connect, &wg)
	time.Sleep(time.Duration(insertInterval) * time.Second)
	wg.Wait()
}

func testTADCleanAfterTheiaMgrResync(t *testing.T, data *TestData) {
	_, jobName1, err := tadrunJob(t, data, "ARIMA", "None")
	require.NoError(t, err)
	_, jobName2, err := tadrunJob(t, data, "ARIMA", "None")
	require.NoError(t, err)

	err = TheiaManagerRestart(t, data, jobName1, "tad")
	require.NoError(t, err)

	// Check the status of jobName2
	stdout, err := tadgetJobStatus(t, data, jobName2)
	require.NoError(t, err)
	assert := assert.New(t)
	assert.Containsf(stdout, "Status of this anomaly detection job is", "stdout: %s", stdout)
	err = data.podWaitForReady(defaultTimeout, jobName2+"-driver", flowVisibilityNamespace)
	require.NoError(t, err)
	_, err = taddeleteJob(t, data, jobName2)
	require.NoError(t, err)

	// Check the SparkApplication and database entries of jobName1 do not exist
	// Allow some time for Theia Manager to delete the stale resources

	err = VerifyJobCleaned(t, data, jobName1, "tadetector", 4)
	require.NoError(t, err)
}
