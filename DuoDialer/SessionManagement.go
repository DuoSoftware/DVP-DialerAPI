package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/fatih/color"
)

func InitiateAgentSessionInfo(company, tenant, sessionExprTime int, campaignId, campaignName, sessionId, number string, integrationData *IntegrationConfig, thirdpartyreference string) {
	companyStr := strconv.Itoa(company)
	tenantStr := strconv.Itoa(tenant)
	sessionExprTimeStr := strconv.Itoa(sessionExprTime)

	data := make(map[string]string)
	data["CompanyId"] = companyStr
	data["TenantId"] = tenantStr
	data["SessionId"] = sessionId
	data["Number"] = number
	data["CampaignId"] = campaignId
	data["CampaignName"] = campaignName
	data["ExpireTime"] = sessionExprTimeStr

	if integrationData != nil {
		intgrData, _ := json.Marshal(*integrationData)
		data["IntegrationData"] = string(intgrData)
	}

	if thirdpartyreference != "" {
		data["ThirdPartyReference"] = thirdpartyreference
	}

	hashKey := fmt.Sprintf("agentSessionInfo:%s:%s", campaignId, sessionId)
	RedisHashSetMultipleField(hashKey, data)
}

//Initiate dial session for a number
func InitiateSessionInfo(company, tenant, sessionExprTime int, sclass, stype, scategory, tryCount, campaignId, scheduleId, campaignName, sessionId, number, reason, dialerStatus, dialTime, serverId string, integrationData *IntegrationConfig, contacts *[]Contact, previewData, thirdpartyreference, businessUnit string) {
	companyStr := strconv.Itoa(company)
	tenantStr := strconv.Itoa(tenant)
	sessionExprTimeStr := strconv.Itoa(sessionExprTime)

	data := make(map[string]string)
	data["Class"] = sclass
	data["Type"] = stype
	data["Category"] = scategory
	data["CompanyId"] = companyStr
	data["TenantId"] = tenantStr
	data["SessionId"] = sessionId
	data["OriginalUuidARDS"] = sessionId
	data["Number"] = number
	data["DialerId"] = dialerId
	data["CampaignId"] = campaignId
	data["ScheduleId"] = scheduleId
	data["CampaignName"] = campaignName
	data["Dialtime"] = dialTime
	data["ChannelCreatetime"] = ""
	data["ChannelAnswertime"] = ""
	data["ServerId"] = serverId
	data["Reason"] = reason
	data["DialerStatus"] = dialerStatus
	data["TryCount"] = tryCount
	data["ExpireTime"] = sessionExprTimeStr
	data["BusinessUnit"] = businessUnit

	tryCountInt, _ := strconv.Atoi(tryCount)

	if tryCountInt > 1 {
		data["CALLBACK"] = "CALLBACK"
	}

	if previewData != "" {
		data["PreviewData"] = previewData
	}

	if thirdpartyreference != "" {
		data["ThirdPartyReference"] = thirdpartyreference
	}

	if integrationData != nil {
		intgrData, _ := json.Marshal(*integrationData)
		data["IntegrationData"] = string(intgrData)
	}

	if contacts != nil {
		data["NumberLoadingMethod"] = "CONTACT"
		if len(*contacts) > 0 {
			contactsByteArr, _ := json.Marshal(*contacts)
			data["Contacts"] = string(contactsByteArr)
		}
	}

	hashKey := fmt.Sprintf("sessionInfo:%s:%s", campaignId, sessionId)
	RedisHashSetMultipleField(hashKey, data)
	PublishEvent(campaignId, sessionId)
	//RedisHashSetNxField(hashKey, "TryCount", tryCount)
}

func SetSessionInfo(campaignId, sessionId, filed, value string) {
	hashKey := fmt.Sprintf("sessionInfo:%s:%s", campaignId, sessionId)
	RedisHashSetField(hashKey, filed, value)
	PublishEvent(campaignId, sessionId)
}

func SetAgentSessionInfo(campaignId, sessionId, filed, value string) {
	hashKey := fmt.Sprintf("agentSessionInfo:%s:%s", campaignId, sessionId)
	RedisHashSetField(hashKey, filed, value)
}

func ManageIntegrationData(sessionInfo map[string]string, integrationType string) {
	defer func() {
		if r := recover(); r != nil {
			color.Red(fmt.Sprintf("Recovered in SendIntegrationData %+v", r))
		}
	}()

	intData := IntegrationConfig{}

	_ = json.Unmarshal([]byte(sessionInfo["IntegrationData"]), &intData)

	fmt.Println(intData)
	bodyData := map[string]interface{}{}

	cyanblue := color.New(color.FgCyan).Add(color.BgMagenta)

	integrationUrl := ""
	dcReason := ""

	if integrationType == "CUSTOMER" {
		for _, element := range intData.Customer.Params {
			if element == "Reason" {
				bodyData[element] = sessionInfo["ReasonCode"]
			} else {
				bodyData[element] = sessionInfo[element]
			}
		}
		integrationUrl = intData.Customer.Url
		dcReason = sessionInfo["ReasonCode"]
		
		cyanblue.Println(fmt.Sprintf("=============CURRENT REASON CUST : %s - dcReason : %s", bodyData["Reason"], dcReason))

		if dcReason == "" {
			dcReason = sessionInfo["Reason"]
			cyanblue.Println(fmt.Sprintf("=============SETTING REASON CUST : %s", dcReason))
		}

	} else if integrationType == "AGENT" {
		for _, element := range intData.Agent.Params {
			if element == "Reason" {
				bodyData[element] = sessionInfo["AgentReasonCode"]
			} else {
				bodyData[element] = sessionInfo[element]
			}

		}
		dcReason = sessionInfo["AgentReasonCode"]
		if dcReason == "" {
			dcReason = sessionInfo["AgentReason"]
			cyanblue.Println(fmt.Sprintf("=============SETTING REASON AGENT : %s", dcReason))
		}
		integrationUrl = intData.Agent.Url

	}

	if integrationUrl != "" {

		if disconnectReasonMap[dcReason] != "" {
			bodyData["Reason"] = disconnectReasonMap[dcReason]
			cyanblue.Println(fmt.Sprintf("=============NEW MAPPED REASON : %s", bodyData["Reason"]))
		}
		
		cyanblue.Println(fmt.Sprintf("=============LAST RESOLVED REASON : %s", bodyData["Reason"]))

		jsonData, _ := json.Marshal(bodyData)

		strdata := string(jsonData)

		cyanblue.Println(fmt.Sprintf("=============SENDING INTEGRATION DATA - URL : %s, Data : %s", integrationUrl, strdata))

		req, err := http.NewRequest("POST", integrationUrl, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err.Error())
		}
		defer resp.Body.Close()

		DialerLog(fmt.Sprintf("response Status:%s", resp.Status))
		DialerLog(fmt.Sprintf("response Headers:%s", resp.Header))
		body, errb := ioutil.ReadAll(resp.Body)
		if errb != nil {
			color.Red(err.Error())
		} else {
			result := string(body)
			DialerLog(fmt.Sprintf("response Body:%s", result))
		}
	} else {
		color.Red("========NO INTEGRATION DATA FOUND=======")
	}

}

func UploadSessionInfo(campaignId, sessionId string) {
	hashKey := fmt.Sprintf("sessionInfo:%s:%s", campaignId, sessionId)
	hashAgentKey := fmt.Sprintf("agentSessionInfo:%s:%s", campaignId, sessionId)
	sessionInfo := RedisHashGetAll(hashKey)
	color.Red(fmt.Sprintf("ARDS UUID : %s", sessionInfo["OriginalUuidARDS"]))
	RedisRemove(hashKey)
	RedisRemove(hashAgentKey)
	dashboardparam2 := "BASIC"
	if sessionInfo["CALLBACK"] == "CALLBACK" {
		dashboardparam2 = "CALLBACK"
	}
	RemoveCampaignCallRealtime(sessionInfo["TenantId"], sessionInfo["CompanyId"], campaignId, sessionId)
	PublishCampaignCallCounts(sessionId, "DISCONNECTED", sessionInfo["CompanyId"], sessionInfo["TenantId"], campaignId, dashboardparam2)
	if sessionInfo["DialerCustomerAnswered"] != "TRUE" {
		PublishCampaignCallCounts(sessionId, "DISCONNECTING", sessionInfo["CompanyId"], sessionInfo["TenantId"], campaignId, dashboardparam2)

		if sessionInfo["IsDialed"] == "TRUE" {
			PublishCampaignCallCounts(sessionId, "REJECTED", sessionInfo["CompanyId"], sessionInfo["TenantId"], campaignId, dashboardparam2)
		}

	}

	//Check Session Is Contact Based Dialing - IF Yes Do Other Operation
	if sessionInfo["Type"] != "SMS" {
		if sessionInfo["NumberLoadingMethod"] == "CONTACT" {
			AddContactToCallback(sessionInfo)
		} else {
			AddPhoneNumberToCallback(sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["TryCount"], sessionInfo["CampaignId"], sessionInfo["ScheduleId"], sessionInfo["Number"], sessionInfo["Reason"], sessionInfo["ArdsCategory"], sessionInfo["ResourceId"], sessionInfo["SessionId"], sessionInfo["ARDSServerType"], sessionInfo["ARDSRequestType"], sessionInfo["ServerId"], sessionInfo)
		}
	}

	PublishEvent(campaignId, sessionId)
	UploadSessionInfoToCampaignManager(sessionInfo)
}

//Clear timed out sessions from campaign..
func ClearTimeoutChannels(campaignId string) {
	sHashKey := fmt.Sprintf("sessionInfo:%s:*", campaignId)
	ongoingSessions := RedisSearchKeys(sHashKey)
	tn := time.Now()
	for _, session := range ongoingSessions {
		sessionInfo := RedisHashGetAll(session)
		company := sessionInfo["CompanyId"]
		tenant := sessionInfo["TenantId"]
		dtime := sessionInfo["Dialtime"]
		ctime := sessionInfo["ChannelCreatetime"]
		atime := sessionInfo["ChannelAnswertime"]
		sid := sessionInfo["ServerId"]
		cid := sessionInfo["CampaignId"]
		sessionid := sessionInfo["SessionId"]
		category := sessionInfo["Category"]
		resourceId := sessionInfo["ResourceId"]
		ardsCategory := sessionInfo["ArdsCategory"]
		expierTimeStr := sessionInfo["ExpireTime"]

		expierTime, _ := strconv.ParseFloat(expierTimeStr, 64)

		dtt, _ := time.Parse(layout4, dtime)
		ctt, _ := time.Parse(layout4, ctime)

		if expierTime > 1 {
			if ctime == "" && tn.Sub(dtt).Seconds() > expierTime {
				if category == "PreviewDial" && resourceId != "" && ardsCategory != "" {
					go ClearResourceSlotWhenReject(company, tenant, ardsCategory, resourceId, sessionid)
				}
				DecrConcurrentChannelCount(sid, cid)
				SetSessionInfo(cid, sessionid, "reason", "ChannelCreate timeout")
				go UploadSessionInfo(cid, sessionid)
			} else if atime == "" && ctime != "" && tn.Sub(ctt).Seconds() > expierTime {
				if category == "PreviewDial" && resourceId != "" && ardsCategory != "" {
					go ClearResourceSlotWhenReject(company, tenant, ardsCategory, resourceId, sessionid)
				}
				DecrConcurrentChannelCount(sid, cid)
				SetSessionInfo(cid, sessionid, "reason", "ChannelAnswer timeout")
				go UploadSessionInfo(cid, sessionid)
			}
		}
	}
}

//func GetSpecificSessionFiled(campaignId, sessionId, field string) string {
//	hashKey := fmt.Sprintf("sessionInfo:%s:%s", campaignId, sessionId)
//	return RedisHashGetField(hashKey, field)
//}

//func GetPhoneNumberAndTryCount(campaignId, sessionId string) (string, int) {
//	hashKey := fmt.Sprintf("sessionInfo:%s:%s", campaignId, sessionId)
//	sessionInfo := RedisHashGetAll(hashKey)
//	number := sessionInfo["Number"]
//	tryCount, _ := strconv.Atoi(sessionInfo["TryCount"])
//	return number, tryCount
//}

//----------------Campaign Manager Service------------------------
func UploadSessionInfoToCampaignManager(sessionInfo map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in UploadSessionInfoToCampaignManager", r)
		}
	}()
	sessionb, err := json.Marshal(sessionInfo)
	if err != nil {
		fmt.Println(err)
		return
	}
	text := string(sessionb)
	fmt.Println(text)
	//upload to campaign service
	serviceurl := fmt.Sprintf("http://%s/DVP/API/1.0.0.0/CampaignManager/Campaign/Session", CreateHost(campaignServiceHost, campaignServicePort))
	jwtToken := fmt.Sprintf("Bearer %s", accessToken)
	internalAuthToken := fmt.Sprintf("%s:%s", sessionInfo["TenantId"], sessionInfo["CompanyId"])

	req, err := http.NewRequest("POST", serviceurl, bytes.NewBuffer(sessionb))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", jwtToken)
	req.Header.Set("companyinfo", internalAuthToken)
	fmt.Println("request:", serviceurl)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer resp.Body.Close()

	body, errb := ioutil.ReadAll(resp.Body)
	//if success remove hashInfo
	if errb != nil {
		fmt.Println(err.Error())
	} else {
		result := string(body)
		fmt.Println("response Body:", result)
	}
}
