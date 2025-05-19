package WebUI

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/free5gc/openapi/models"
	"github.com/free5gc/util/mongoapi"
	"github.com/free5gc/webconsole/backend/logger"
	"github.com/free5gc/webconsole/backend/webui_context"
)

func GetUEUsage(c *gin.Context) {
	logger.BillingLog.Info("Get UE Usage")
	setCorsHeader(c)

	webui := webui_context.GetSelf()
	webui.UpdateNfProfiles()

	var uesJsonData interface{}
	if amfUris := webui.GetOamUris(models.NrfNfManagementNfType_AMF); amfUris != nil {
		requestUri := fmt.Sprintf("%s/namf-oam/v1/registered-ue-context", amfUris[0])

		ctx, _, err := webui.GetTokenCtx(models.ServiceName_NAMF_OAM, models.NrfNfManagementNfType_AMF)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token context error"})
			return
		}

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, requestUri, nil)
		webui.RequestBindToken(req, ctx)

		resp, err := httpsClient.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Request to AMF failed"})
			return
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&uesJsonData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Decode UE data failed"})
			return
		}
	}

	uesBsonA := toBsonA(uesJsonData)
	usageRecords := make([]RatingGroupDataUsage, 0)

	type OfflineSliceTypeMap struct {
		supi           string
		snssai         string
		dnn            string
		unitcost       int64
		flowTotalVolum int64
		flowTotalUsage int64
	}
	offlineChargingSliceTypeMap := make(map[string]OfflineSliceTypeMap)

	for _, ueData := range uesBsonA {
		ueBsonM := toBsonM(ueData)
		supi := ueBsonM["Supi"].(string)

		rgDataUsages, err := parseCDR(supi)
		if err != nil {
			continue
		}

		for rg, du := range rgDataUsages {
			filter := bson.M{"ueId": supi, "ratingGroup": rg}
			chargingDataInterface, err := mongoapi.RestfulAPIGetOne(chargingDataColl, filter)
			if err != nil || len(chargingDataInterface) == 0 {
				continue
			}

			var chargingData ChargingData
			_ = json.Unmarshal(mapToByte(chargingDataInterface), &chargingData)

			switch chargingData.ChargingMethod {
			case ChargingOffline:
				unitcost, _ := strconv.ParseInt(chargingData.UnitCost, 10, 64)
				if unitcost == 0 {
					unitcost = 1
				}
				key := chargingData.UeId + chargingData.Snssai
				pdu_level := offlineChargingSliceTypeMap[key]

				if chargingData.Filter != "" {
					du.Usage = du.TotalVol * unitcost
					pdu_level.flowTotalUsage += du.Usage
					pdu_level.flowTotalVolum += du.TotalVol
				} else {
					pdu_level.supi = chargingData.UeId
					pdu_level.snssai = chargingData.Snssai
					pdu_level.dnn = chargingData.Dnn
					pdu_level.unitcost = unitcost
				}
				offlineChargingSliceTypeMap[key] = pdu_level
			case ChargingOnline:
				q, _ := strconv.Atoi(chargingData.Quota)
				du.QuotaLeft = int64(q)
			}

			du.Snssai = chargingData.Snssai
			du.Dnn = chargingData.Dnn
			du.Supi = supi
			du.Filter = chargingData.Filter

			rgDataUsages[rg] = du
			usageRecords = append(usageRecords, du)
		}
	}

	for i, rd := range usageRecords {
		if rd.Filter != "" {
			continue
		}
		key := rd.Supi + rd.Snssai
		if val, ok := offlineChargingSliceTypeMap[key]; ok {
			rd.Usage += val.flowTotalUsage
			rd.Usage += (rd.TotalVol - val.flowTotalVolum) * val.unitcost
			usageRecords[i] = rd
		}
	}

	c.JSON(http.StatusOK, usageRecords)
}
