package elastic

import (
	"context"
	"errors"
	"fmt"
	"github.com/lruggieri/magneticoi/pkg/queue"
	"github.com/lruggieri/magneticoi/pkg/util"
	"github.com/olivere/elastic/v7"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"
)

type Connection struct {
	Connection *elastic.Client
}

const(
	bulkSize          = 1000
	IndexNameTorrents = "torrents"
	InsertionTickTime = 5 * time.Second
)


type indexBlocks struct{
	templatePath []string
	insertionFunction func(iElasticClient *elastic.Client, iIndexName string, iDbChannel chan queue.ExpandedTorrentSummary) (oErr error)
}

//maps required index => path to relative mapping (path elements to be joined)
var requiredIndices = map[string]indexBlocks{
	IndexNameTorrents: {
		templatePath:      []string{util.GetCallerPaths(1)[0],"mappings", "torrentSummary.json"},
		insertionFunction: InsertTorrentSummary,
	},
}

func checkEsDB(iElasticClient *elastic.Client)(oError error){
	for requiredIndexName, requiredIndexBlocks := range requiredIndices {
		indexExist, err := iElasticClient.IndexExists(requiredIndexName).Do(context.Background())
		if err != nil {
			return err
		}
		if !indexExist {
			//index not found, we have to create it

			util.Logger.Info("creating index " + requiredIndexName)
			indexTemplateFile, err := os.Open(path.Join(requiredIndexBlocks.templatePath...))
			if err != nil {
				return err
			}

			indexTemplateBytes, err := ioutil.ReadAll(indexTemplateFile)
			if err != nil {
				return err
			}

			util.Logger.Info("using template " + string(indexTemplateBytes))

			indexCreationResult, err := iElasticClient.CreateIndex(requiredIndexName).BodyString(string(indexTemplateBytes)).Do(context.Background())
			if err != nil {
				return err
			}
			if !indexCreationResult.Acknowledged {
				return errors.New(requiredIndexName + " index creation not acknowledged")
			}
		}
	}

	return nil
}

func InsertTorrentSummary(iElasticClient *elastic.Client, iIndexName string, iDbChannel chan queue.ExpandedTorrentSummary) (oErr error){
	bulkRequest := iElasticClient.Bulk()
	currentBulkElements := 0

	var lastInsertionTime = time.Now()
	commitBulk := func (bulkRequest *elastic.BulkService, bulkElements *int) (oErr error){
		defer func(){
			lastInsertionTime = time.Now()
		}()

		if *bulkElements > 0{
			bulkResponse, err := bulkRequest.Do(context.Background())
			if err != nil{
				return err
			}

			if bulkResponse.Errors{
				util.Logger.Error("errors during bulk. Failed elements: ",len(bulkResponse.Failed()))
				for _,fe := range bulkResponse.Failed(){
					fmt.Println("\t"+fe.Error.Reason)
				}
			}

			succeeded := bulkResponse.Succeeded()
			indexed := bulkResponse.Indexed()
			updated := bulkResponse.Updated()
			created := bulkResponse.Created()
			failed := bulkResponse.Failed()

			util.Logger.Info("processed "+strconv.Itoa(*bulkElements)+" elements")

			util.Logger.Info(
				"\tsucceeded:" + strconv.Itoa(len(succeeded)) +
					"\n\tindexed:" + strconv.Itoa(len(indexed)) +
					"\n\tupdated:" + strconv.Itoa(len(updated)) +
					"\n\tcreated:" + strconv.Itoa(len(created)) +
					"\n\tfailed:" + strconv.Itoa(len(failed)),
			)

			bulkRequest.Reset()
			*bulkElements = 0
		}

		return nil
	}

	//inserting elements either every 5 seconds or when bulkSize elements are queued
	for{
		select{
		case <- time.Tick(InsertionTickTime):{
			if currentBulkElements > 0{
				err := commitBulk(bulkRequest,&currentBulkElements)
				if err != nil{
					return err
				}
			}else{
				util.Logger.Warn("no torrents to insert")
			}
		}
		case tSummary := <- iDbChannel:{
			bulkRequest.Add(elastic.NewBulkUpdateRequest().Index(iIndexName).Id(tSummary.InfoHash).Doc(tSummary).DocAsUpsert(true))
			currentBulkElements++

			if time.Now().Sub(lastInsertionTime) > InsertionTickTime || (currentBulkElements > 0 && currentBulkElements % bulkSize == 0){
				err := commitBulk(bulkRequest,&currentBulkElements)
				if err != nil{
					return err
				}
			}
		}
		}
	}
}

func New(iHost, iPort string) (oElasticDB *Connection, oErr error){
	esUrl := iHost
	if len(iPort) > 0 {
		esUrl += ":" + iPort
	}
	es, err := elastic.NewClient(elastic.SetSniff(false),elastic.SetURL(esUrl))
	if err != nil{
		return nil, err
	}

	err = checkEsDB(es)
	if err != nil{
		return nil, err
	}

	return &Connection{Connection: es}, nil
}