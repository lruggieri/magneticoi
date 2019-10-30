package elastic

import (
	"context"
	"errors"
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
	connection *elastic.Client
}

const(
	bulkSize          = 1000
	IndexNameResource = "resource"
)


type indexBlocks struct{
	templatePath []string
	insertionFunction func(iElasticClient *elastic.Client, iIndexName string, iDbChannel chan queue.SimpleTorrentSummary) (oErr error)
}

//maps required index => path to relative mapping (path elements to be joined)
var requiredIndices = map[string]indexBlocks{
	IndexNameResource: {
		templatePath:      []string{util.GetCallerPaths(1)[0],"mappings", "simpleTorrentSummary.json"},
		insertionFunction: insertTorrentSummary,
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

func insertTorrentSummary(iElasticClient *elastic.Client, iIndexName string, iDbChannel chan queue.SimpleTorrentSummary) (oErr error){
	bulkRequest := iElasticClient.Bulk()
	currentBulkElements := 0
	totalInsertion := 0

	commitBulk := func (bulkRequest *elastic.BulkService, bulkElements *int, totalInsertion *int) (oErr error){
		bulkResponse, err := bulkRequest.Do(context.Background())
		if err != nil{
			return err
		}

		indexed := bulkResponse.Indexed()
		if len(indexed) != *bulkElements{
			return errors.New("tried to index "+strconv.Itoa(*bulkElements)+" but " +
				"successfully indexed "+strconv.Itoa(len(indexed)))
		}

		*totalInsertion += *bulkElements
		util.Logger.Info("inserted:" + strconv.Itoa(*totalInsertion) + " resources")

		bulkRequest.Reset()
		*bulkElements = 0

		return nil
	}

	for{
		select{
		case <- time.Tick(5 * time.Second):{
			err := commitBulk(bulkRequest,&currentBulkElements, &totalInsertion)
			if err != nil{
				return err
			}
		}
		case tSummary := <- iDbChannel:{
			bulkRequest.Add(elastic.NewBulkIndexRequest().Index(iIndexName).Id(tSummary.InfoHash).Doc(tSummary))
			currentBulkElements++

			if currentBulkElements > 0 && currentBulkElements % bulkSize == 0{
				err := commitBulk(bulkRequest,&currentBulkElements, &totalInsertion)
				if err != nil{
					return err
				}
			}
		}
		}
	}
}