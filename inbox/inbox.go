package inbox

import (
	"encoding/json"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"

	"github.com/cloudfoundry-incubator/stager/outbox"
	"github.com/cloudfoundry-incubator/stager/stager"
)

const DiegoStageStartSubject = "diego.staging.start"

type Inbox struct {
	natsClient      yagnats.NATSClient
	stager          stager.Stager
	validateRequest RequestValidator

	logger *steno.Logger
}

type RequestValidator func(models.StagingRequestFromCC) error

func Listen(natsClient yagnats.NATSClient, stager stager.Stager, validator RequestValidator, logger *steno.Logger) {
	inbox := Inbox{
		natsClient:      natsClient,
		stager:          stager,
		validateRequest: validator,

		logger: logger,
	}

	inbox.Listen()
}

func (inbox *Inbox) Listen() {
	for {
		_, err := inbox.natsClient.SubscribeWithQueue(DiegoStageStartSubject, "diego.stagers", func(message *yagnats.Message) {
			stagingRequest := models.StagingRequestFromCC{}

			err := json.Unmarshal(message.Payload, &stagingRequest)
			if err != nil {
				inbox.logError("staging.request.malformed", err, message)
				return
			}

			err = inbox.validateRequest(stagingRequest)
			if err != nil {
				inbox.logError("staging.request.invalid", err, message)
				inbox.sendErrorResponse("Invalid staging request: "+err.Error(), stagingRequest)
				return
			}

			inbox.logger.Infod(
				map[string]interface{}{
					"message": stagingRequest,
				},
				"staging.request.received",
			)

			err = inbox.stager.Stage(stagingRequest)
			if err != nil {
				inbox.logError("stager.staging.failed", err, stagingRequest)
				inbox.sendErrorResponse("Staging failed: "+err.Error(), stagingRequest)
				return
			}
		})

		if err == nil {
			time.Sleep(500 * time.Millisecond)
			break
		}
	}
}

func (inbox *Inbox) logError(logMessage string, err error, message interface{}) {
	inbox.logger.Errord(map[string]interface{}{
		"message": message,
		"error":   err.Error(),
	}, logMessage)
}

func (inbox *Inbox) sendErrorResponse(errorMessage string, request models.StagingRequestFromCC) {
	response := models.StagingResponseForCC{
		AppId:  request.AppId,
		TaskId: request.TaskId,
		Error:  errorMessage,
	}

	if responseJson, err := json.Marshal(response); err == nil {
		inbox.natsClient.Publish(outbox.DiegoStageFinishedSubject, responseJson)
	}
}
