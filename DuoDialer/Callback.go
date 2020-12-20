package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/fatih/color"
)

//----------Campaign Manager Service-----------------------
func RequestCampaignCallbackConfig(tenant, company, configureId int) ([]CampaignCallbackInfo, bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in RequestCampaignCallbackConfig", r)
		}
	}()
	//Request campaign from Campaign Manager service
	campaignCallbackInfo := make([]CampaignCallbackInfo, 0)
	jwtToken := fmt.Sprintf("Bearer %s", accessToken)
	internalAuthToken := fmt.Sprintf("%d:%d", tenant, company)
	client := &http.Client{}

	request := fmt.Sprintf("http://%s/DVP/API/1.0.0.0/CampaignManager/Campaign/Configuration/%d/all", CreateHost(campaignServiceHost, campaignServicePort), configureId)
	DialerLog(fmt.Sprintf("Start RequestCampaignCallbackConfig request: %s", request))
	req, _ := http.NewRequest("GET", request, nil)
	req.Header.Set("authorization", jwtToken)
	req.Header.Set("companyinfo", internalAuthToken)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return campaignCallbackInfo, false
	}
	defer resp.Body.Close()

	response, _ := ioutil.ReadAll(resp.Body)

	var campaignCallbackResult CampaignCallbackInfoResult
	json.Unmarshal(response, &campaignCallbackResult)
	if campaignCallbackResult.IsSuccess == true {
		if len(campaignCallbackResult.Result) > 0 {
			camRes := campaignCallbackResult.Result[0]
			for _, config := range camRes.CampCallbackConfigurations {
				campaignCallbackInfo = append(campaignCallbackInfo, config)
			}
			return campaignCallbackInfo, camRes.AllowCallBack
		}
	}
	return campaignCallbackInfo, false
}

//----------CallbackServer Self Host-----------------------
func UploadCallbackInfo(company, tenant int, callbackTime time.Time, campaignId, cbClass, cbType, cbCategory, cbUrl, cbObj string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in UploadCallbackInfo", r)
		}
	}()

	magentawhite := color.New(color.FgMagenta).Add(color.BgWhite)
	magentawhite.Println("(17) Start UploadCallbackInfo")

	fmt.Println("Start UploadCallbackInfo")
	callback := CampaignCallback{}

	callback.Company = company
	callback.Tenant = tenant
	callback.Class = cbClass
	callback.Type = cbType
	callback.Category = cbCategory
	callback.DialoutTime = callbackTime
	callback.CallbackUrl = cbUrl
	callback.CallbackObj = cbObj
	callback.CampaignId = campaignId

	jsonData, _ := json.Marshal(callback)

	serviceurl := fmt.Sprintf("http://%s/DVP/API/1.0.0.0/Callback/AddCallback", CreateHost(callbackServerHost, callbackServerPort))
	jwtToken := fmt.Sprintf("Bearer %s", accessToken)
	internalAuthToken := fmt.Sprintf("%d:%d", tenant, company)
	req, err := http.NewRequest("POST", serviceurl, bytes.NewBuffer(jsonData))

	fmt.Println("Add callback data:: ", serviceurl, "::", string(jsonData))

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

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, errb := ioutil.ReadAll(resp.Body)
	if errb != nil {
		fmt.Println(err.Error())
	} else {
		result := string(body)
		fmt.Println("response Body:", result)
	}
}

//----------Campaign Callback Info-----------------------
func AddCampaignCallbackConfigInfo(company, tenant, campaignId, configureId int) {
	callbackConfigList, AllowCallback := RequestCampaignCallbackConfig(tenant, company, configureId)
	allowCallbackKey := fmt.Sprintf("CampaignAllowCallback:%d:%d:%d", company, tenant, campaignId)
	isSet := RedisSet(allowCallbackKey, strconv.FormatBool(AllowCallback))
	if isSet == "OK" {
		for _, callbackConfig := range callbackConfigList {
			hashKey := fmt.Sprintf("CampaignCallbackConfig:%d:%d:%d:%s", company, tenant, campaignId, callbackConfig.CampCallBackReasons.Reason)
			data := make(map[string]string)
			data["MaxCallbackCount"] = strconv.Itoa(callbackConfig.MaxCallBackCount)
			data["CallbackInterval"] = strconv.Itoa(callbackConfig.CallBackInterval)
			RedisHashSetMultipleField(hashKey, data)
		}
	}
}

//func GetMaxCallbackCount(company, tenant int, campaignId, reason string) int {
//	hashKey := fmt.Sprintf("CampaignCallbackConfig:%d:%d:%s:%s", company, tenant, campaignId, reason)
//	callbackCountStr := RedisHashGetField(hashKey, "MaxCallbackCount")
//	maxCallbackCount, _ := strconv.Atoi(callbackCountStr)
//	return maxCallbackCount
//}

//func GetCallbackInterval(company, tenant int, campaignId, reason string) int {
//	hashKey := fmt.Sprintf("CampaignCallbackConfig:%d:%d:%s:%s", company, tenant, campaignId, reason)
//	callbackIntervalStr := RedisHashGetField(hashKey, "CallbackInterval")
//	callbackInterval, _ := strconv.Atoi(callbackIntervalStr)
//	return callbackInterval
//}

func GetCallbackDetails(company, tenant int, campaignId, reason string) (maxCallbackCount, callbackInterval int, isReasonExists bool) {
	hashKey := fmt.Sprintf("CampaignCallbackConfig:%d:%d:%s:%s", company, tenant, campaignId, reason)
	isReasonExists = RedisCheckKeyExist(hashKey)
	if isReasonExists {
		callbackIntervalStr := RedisHashGetAll(hashKey)
		callbackInterval, _ := strconv.Atoi(callbackIntervalStr["CallbackInterval"])
		maxCallbackCount, _ := strconv.Atoi(callbackIntervalStr["MaxCallbackCount"])
		return maxCallbackCount, callbackInterval, isReasonExists
	}
	return 0, 0, isReasonExists
}

func GetAllowCallback(company, tenant int, campaignId string) bool {
	hashKey := fmt.Sprintf("CampaignAllowCallback:%d:%d:%s", company, tenant, campaignId)
	fmt.Println("Key:GetAllowCallback:: ", hashKey)
	callbackIntervalStr := RedisGet(hashKey)
	fmt.Println("Result:GetAllowCallback:: ", callbackIntervalStr)
	allowCallback, _ := strconv.ParseBool(callbackIntervalStr)
	return allowCallback
}

func RemoveCampaignCallbackConfigInfo(company, tenant int, campaignId string) {
	searchKey := fmt.Sprintf("CampaignCallbackConfig:%d:%d:%s:*", company, tenant, campaignId)
	allowCallbackKey := fmt.Sprintf("CampaignAllowCallback:%d:%d:%s", company, tenant, campaignId)
	RedisRemove(allowCallbackKey)
	callbackKeys := RedisSearchKeys(searchKey)
	for _, key := range callbackKeys {
		RedisRemove(key)
	}
}

func ValidateDisconnectReason(disconnectReason string) (keyExist bool, value string) {
	confKey := fmt.Sprintf("CallbackReason:%s", disconnectReason)
	value = RedisGet(confKey)

	if value == "" {
		keyExist = false
	} else {
		keyExist = true
	}
	return
}

/* func AddContactToCallback(sessionInfo map[string]string) {
	DialerLog("start AddPhoneNumberToCallback")

	color.Cyan(fmt.Sprintf("SESSION : %v", sessionInfo))

	contactsList := []Contact{}

	_ = json.Unmarshal([]byte(sessionInfo["Contacts"]), &contactsList)

	if contactsList != nil && len(contactsList) > 0 {
		company, _ := strconv.Atoi(sessionInfo["CompanyId"])
		tenant, _ := strconv.Atoi(sessionInfo["TenantId"])
		campId, _ := strconv.Atoi(sessionInfo["CampaignId"])
		scheduleId, _ := strconv.Atoi(sessionInfo["ScheduleId"])
		isdisconnectReasonAllowed, hangupGruop := ValidateDisconnectReason(sessionInfo["Reason"])
		DialerLog(fmt.Sprintf("isdisconnectReasonAllowed:: %t", isdisconnectReasonAllowed))
		DialerLog(fmt.Sprintf("hangupGruop:: %s", hangupGruop))
		if isdisconnectReasonAllowed {

			campaignInfo, isCamExists := GetCampaign(company, tenant, campId)
			DialerLog(fmt.Sprintf("isCamExists:: %t", isCamExists))

			if isCamExists {

				scheduleInfo := CampaignShedule{}
				defaultScheduleInfo := CampaignShedule{}
				for _, schedule := range campaignInfo.CampScheduleInfo {
					if schedule.ScheduleId == scheduleId {
						scheduleInfo = schedule
						break
					}
				}

				if scheduleInfo != defaultScheduleInfo {

					location, _ := time.LoadLocation(scheduleInfo.TimeZone)

					tmNow := time.Now().In(location)
					callbackTime := time.Date(tmNow.Year(), tmNow.Month(), tmNow.Day(), tmNow.Hour(), tmNow.Minute(), tmNow.Second(), 0, location)
					fmt.Println("callbackTime:: ", callbackTime)

					//tempCampaignEndDate, _ := time.Parse(layout2, campaignInfo.CampConfigurations.EndDate)
					//campaignEndDate := time.Date(tempCampaignEndDate.Year(), tempCampaignEndDate.Month(), tempCampaignEndDate.Day(), tempCampaignEndDate.Hour(), tempCampaignEndDate.Minute(), tempCampaignEndDate.Second(), 0, location)

					scheduleEndDate := scheduleInfo.EndDate

					fmt.Println("Callback:scheduleEndDate:: ", scheduleEndDate)
					if scheduleEndDate.After(callbackTime) {
						fmt.Println("Start to build CallbackInfo")
						//scheduleIdStr := strconv.Itoa(scheduleInfo.ScheduleId)
						validateAppoinment := CheckAppoinmentForCallback(company, tenant, sessionInfo["ScheduleId"], callbackTime, scheduleInfo.TimeZone)
						fmt.Println("validateAppoinmentFor Callback:: ", validateAppoinment)
						if validateAppoinment {
							//HERE YOU HAVE TO ADD CALLBACK NUMBER TO THE FRONT OF THE NUMBER QUEUE AND RE ADJUST THE CONTACTS ARRAY
							nextContactNum := contactsList[0]
							r := contactsList[1:]
							contactDet := ContactsDetails{Phone: nextContactNum.Contact, Api_Contacts: r, PreviewData: sessionInfo["PreviewData"]}
							AddContactToFront(company, tenant, sessionInfo["CampaignId"], contactDet)
							//go UploadCallbackInfo(_company, _tenant, callbackTime, campaignId, "DIALER", "CALLBACK", "INTERNAL", cbUrl, string(jsonData))
						}
					}
				} else {
					fmt.Println("Add Callback Failed, No Valied Schedule Found")
				}

			} else {
				fmt.Println("Add Callback Failed, No Existing Campaign Found")
			}
		}
	} else {
		color.Magenta("NO CONTACTS FOUND FOR RE DIALING")

	}

} */

func RedialContactToSameAgent(campaignInfo Campaign, sessionInfo map[string]string, customerNumber string) {
	magentawhite := color.New(color.FgMagenta).Add(color.BgWhite)
	magentawhite.Println("(28) Redial Contact to Same Agent")
	resourceServerInfos := GetResourceServerInfo(campaignInfo.CompanyId, campaignInfo.TenantId, "*", campaignInfo.CampaignChannel)

	uuid := GetUuid(resourceServerInfos.Url)
	internalAuthToken := fmt.Sprintf("%d:%d", campaignInfo.TenantId, campaignInfo.CompanyId)
	context := fmt.Sprintf("%d_%d_dialer", campaignInfo.TenantId, campaignInfo.CompanyId)
	trunkCode, ani, dnis, xGateway := GetTrunkCode(internalAuthToken, campaignInfo.CampConfigurations.Caller, customerNumber, context, "")

	if sessionInfo["OriginalUuidARDS"] == "" {
		magentawhite.Println("NO ORIGINAL UUID ARDS FOUND - SETTING CURRENT UUID")
		sessionInfo["OriginalUuidARDS"] = sessionInfo["SessionId"]
	} else {
		magentawhite.Println("UUID ARDS FOUND - EXITING")
	}
	sessionInfo["SessionId"] = uuid
	sessionInfo["Number"] = dnis

	hashKey := fmt.Sprintf("sessionInfo:%s:%s", sessionInfo["CampaignId"], uuid)
	RedisHashSetMultipleField(hashKey, sessionInfo)

	companyInt := campaignInfo.CompanyId
	tenantInt := campaignInfo.TenantId
	resourceServer := GetResourceServerInfo(companyInt, tenantInt, resourceServerInfos.ResourceServerId, campaignInfo.CampaignChannel)

	customCompanyStr := fmt.Sprintf("%d_%d", companyInt, tenantInt)

	if trunkCode != "" && dnis != "" && sessionInfo["Extention"] != "" {

		magentawhite.Println("(29) Rule Found")

		var param string
		var furl string
		var data string
		var dial bool

		dial = true

		ardsQueueName := sessionInfo["ArdsQueueName"]
		param = fmt.Sprintf("{sip_h_DVP-DESTINATION-TYPE=GATEWAY,DVP_CALL_DIRECTION=outbound,sip_h_X-Gateway=%s,ards_skill_display=%s,DVP_CUSTOM_PUBID=%s,nolocal:DIALER_AGENT_EVENT=%s,CustomCompanyStr=%s,CampaignId=%s,CampaignName='%s',tenantid=%s,companyid=%s,ards_resource_id=%s,ards_client_uuid=%s,origination_uuid=%s,ards_servertype=%s,ards_requesttype=%s,DVP_ACTION_CAT=DIALER,DVP_OPERATION_CAT=AGENT,return_ring_ready=false,ignore_early_media=true,origination_caller_id_number=%s,origination_caller_id_name=%s,DialerCustomerNumber=%s,DialerAgentName=%s,DialerAgentSipName=%s,CALL_LEG_TYPE=CUSTOMER,OriginalUuidARDS=%s}", xGateway, ardsQueueName, subChannelName, subChannelNameAgent, customCompanyStr, sessionInfo["CampaignId"], sessionInfo["CampaignName"], sessionInfo["TenantId"], sessionInfo["CompanyId"], sessionInfo["ResourceId"], uuid, uuid, sessionInfo["ARDSServerType"], sessionInfo["ARDSRequestType"], ani, ani, dnis, sessionInfo["Agent"], sessionInfo["AgentSipName"], sessionInfo["OriginalUuidARDS"])
		furl = fmt.Sprintf("sofia/gateway/%s/%s", trunkCode, dnis)

		//call recording enable
		data = fmt.Sprintf(" %s xml dialer", sessionInfo["AgentExtension"])
		//data = fmt.Sprintf(" %s xml dialer", phoneNumber)

		if dial == true {
			SetSessionInfo(sessionInfo["CampaignId"], uuid, "Reason", "Dial Number")
			redwhite := color.New(color.FgRed).Add(color.BgWhite)
			redwhite.Println(fmt.Sprintf("DIALING OUT CALL - AGENT CAMPAIGN : %s | NUMBER : %s", sessionInfo["CampaignName"], dnis))

			//resp, err := DialNew(resourceServer.Url, param, furl, data)
			//PublishCampaignCallCounts(uuid, "DIALED", sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["CampaignId"])
			SetSessionInfo(sessionInfo["CampaignId"], uuid, "IsDialed", "TRUE")
			SetSessionInfo(sessionInfo["CampaignId"], uuid, "CALLBACK", "CALLBACK")
			AddCampaignCallsRealtime(dnis, sessionInfo["TryCount"], "DIALING", sessionInfo["TenantId"], sessionInfo["CompanyId"], sessionInfo["CampaignId"], uuid)
			//UpdateCampaignCallRealtimeField("DialState", "DIALING", sessionInfo["TenantId"], sessionInfo["CompanyId"], sessionInfo["CampaignId"], uuid)
			PublishCampaignCallCounts(uuid, "DIALING", sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["CampaignId"], "CALLBACK")
			resp, err := Dial(resourceServer.Url, param, furl, data)
			HandleDialResponse(resp, err, resourceServer, sessionInfo["CampaignId"], uuid)
		} else {
			SetSessionInfo(sessionInfo["CampaignId"], uuid, "Reason", "Invalied ContactType")
			SetSessionInfo(sessionInfo["CampaignId"], uuid, "DialerStatus", "dial_failed")
			go UploadSessionInfo(sessionInfo["CampaignId"], uuid)
			//AgentReject(sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["CampaignId"], uuid, sessionInfo["ARDSRequestType"], sessionInfo["ResourceId"], "Invalied ContactType")
		}
	} else {
		magentawhite.Println("(30) Rule Not Found - Releasing agent")
		SetSessionInfo(sessionInfo["CampaignId"], uuid, "Reason", "dial_failed")
		SetSessionInfo(sessionInfo["CampaignId"], uuid, "DialerStatus", "dial_failed")
		go UploadSessionInfo(sessionInfo["CampaignId"], uuid)
		//RemoveRequest(sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["OriginalUuidARDS"])
	}
}

func SetNextContact(contactsList []Contact, sessionInfo map[string]string) {

	magentawhite := color.New(color.FgMagenta).Add(color.BgWhite)

	isdisconnectReasonAllowed, _ := ValidateDisconnectReason(sessionInfo["Reason"])

	DialerLog(fmt.Sprintf("Disconnection reason is : %s", sessionInfo["Reason"]))

	if isdisconnectReasonAllowed {
		if contactsList != nil && len(contactsList) > 0 {
			magentawhite.Println("(18) Contacts found")
			company, _ := strconv.Atoi(sessionInfo["CompanyId"])
			tenant, _ := strconv.Atoi(sessionInfo["TenantId"])
			campId, _ := strconv.Atoi(sessionInfo["CampaignId"])
			scheduleId, _ := strconv.Atoi(sessionInfo["ScheduleId"])

			campaignInfo, isCamExists := GetCampaign(company, tenant, campId)
			DialerLog(fmt.Sprintf("isCamExists:: %t", isCamExists))

			if isCamExists {

				magentawhite.Println("(19) Campaign Exists")

				scheduleInfo := CampaignShedule{}
				defaultScheduleInfo := CampaignShedule{}
				for _, schedule := range campaignInfo.CampScheduleInfo {
					if schedule.ScheduleId == scheduleId {
						scheduleInfo = schedule
						break
					}
				}

				if scheduleInfo != defaultScheduleInfo {

					magentawhite.Println("(20) Schedule Exists")

					location, _ := time.LoadLocation(scheduleInfo.TimeZone)

					tmNow := time.Now().In(location)
					callbackTime := time.Date(tmNow.Year(), tmNow.Month(), tmNow.Day(), tmNow.Hour(), tmNow.Minute(), tmNow.Second(), 0, location)
					fmt.Println("callbackTime:: ", callbackTime)

					//tempCampaignEndDate, _ := time.Parse(layout2, campaignInfo.CampConfigurations.EndDate)
					//campaignEndDate := time.Date(tempCampaignEndDate.Year(), tempCampaignEndDate.Month(), tempCampaignEndDate.Day(), tempCampaignEndDate.Hour(), tempCampaignEndDate.Minute(), tempCampaignEndDate.Second(), 0, location)

					scheduleEndDate := scheduleInfo.EndDate

					fmt.Println("Callback:scheduleEndDate:: ", scheduleEndDate)
					if scheduleEndDate.After(callbackTime) {
						magentawhite.Println("(21) Schedule Is Ok to Start")
						fmt.Println("Start to build CallbackInfo")
						//scheduleIdStr := strconv.Itoa(scheduleInfo.ScheduleId)
						validateAppoinment := CheckAppoinmentForCallback(company, tenant, sessionInfo["ScheduleId"], callbackTime, scheduleInfo.TimeZone)
						fmt.Println("validateAppoinmentFor Callback:: ", validateAppoinment)
						if validateAppoinment {
							magentawhite.Println("(22) Valid Appointment")
							//HERE YOU HAVE TO ADD CALLBACK NUMBER TO THE FRONT OF THE NUMBER QUEUE AND RE ADJUST THE CONTACTS ARRAY
							nextContactNum := contactsList[0]
							r := contactsList[1:]
							contactsByteArr, _ := json.Marshal(r)
							sessionInfo["Contacts"] = string(contactsByteArr)
							sessionInfo["TryCount"] = "1"
							//contactDet := ContactsDetails{Phone: nextContactNum.Contact, Api_Contacts: r, PreviewData: sessionInfo["PreviewData"]}
							RedialContactToSameAgent(campaignInfo, sessionInfo, nextContactNum.Contact)
							//AddContactToFront(company, tenant, sessionInfo["CampaignId"], contactDet)
							//go UploadCallbackInfo(_company, _tenant, callbackTime, campaignId, "DIALER", "CALLBACK", "INTERNAL", cbUrl, string(jsonData))
						} else {
							//RELEASING AGENT
							magentawhite.Println("(23) Release Agent")
							SendAgentIntegrationData(sessionInfo)
							SetAgentStatusArds(sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["ArdsCategory"], sessionInfo["ResourceId"], sessionInfo["OriginalUuidARDS"], "Completed", sessionInfo["ARDSServerType"], sessionInfo["ARDSRequestType"])
							DecrConcurrentChannelCount(sessionInfo["ServerId"], sessionInfo["CampaignId"])
						}
					} else {
						//RELEASING AGENT
						magentawhite.Println("(24) Release Agent")
						SendAgentIntegrationData(sessionInfo)
						SetAgentStatusArds(sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["ArdsCategory"], sessionInfo["ResourceId"], sessionInfo["OriginalUuidARDS"], "Completed", sessionInfo["ARDSServerType"], sessionInfo["ARDSRequestType"])
						DecrConcurrentChannelCount(sessionInfo["ServerId"], sessionInfo["CampaignId"])
					}
				} else {
					fmt.Println("Add Callback Failed, No Valied Schedule Found")
					magentawhite.Println("(25) Add Callback Failed, No Valied Schedule Found")
					//RELEASING AGENT
					SendAgentIntegrationData(sessionInfo)
					SetAgentStatusArds(sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["ArdsCategory"], sessionInfo["ResourceId"], sessionInfo["OriginalUuidARDS"], "Completed", sessionInfo["ARDSServerType"], sessionInfo["ARDSRequestType"])
					DecrConcurrentChannelCount(sessionInfo["ServerId"], sessionInfo["CampaignId"])
				}

			} else {
				fmt.Println("Add Callback Failed, No Existing Campaign Found")
				magentawhite.Println("(26) Add Callback Failed, No Existing Campaign Found")
				//RELEASING AGENT
				SendAgentIntegrationData(sessionInfo)
				SetAgentStatusArds(sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["ArdsCategory"], sessionInfo["ResourceId"], sessionInfo["OriginalUuidARDS"], "Completed", sessionInfo["ARDSServerType"], sessionInfo["ARDSRequestType"])
				DecrConcurrentChannelCount(sessionInfo["ServerId"], sessionInfo["CampaignId"])
			}
		} else {
			color.Magenta("NO CONTACTS FOUND FOR RE DIALING")
			magentawhite.Println("(27) NO CONTACTS FOUND FOR RE DIALING")
			//RELEASING AGENT
			SendAgentIntegrationData(sessionInfo)
			SetAgentStatusArds(sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["ArdsCategory"], sessionInfo["ResourceId"], sessionInfo["OriginalUuidARDS"], "Completed", sessionInfo["ARDSServerType"], sessionInfo["ARDSRequestType"])
			DecrConcurrentChannelCount(sessionInfo["ServerId"], sessionInfo["CampaignId"])

		}

	} else {
		magentawhite.Println("(28) DISCONNECT REASON NOT SET")
		//RELEASING AGENT
		SendAgentIntegrationData(sessionInfo)
		SetAgentStatusArds(sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["ArdsCategory"], sessionInfo["ResourceId"], sessionInfo["OriginalUuidARDS"], "Completed", sessionInfo["ARDSServerType"], sessionInfo["ARDSRequestType"])
		DecrConcurrentChannelCount(sessionInfo["ServerId"], sessionInfo["CampaignId"])
	}

}

func AddContactToCallback(sessionInfo map[string]string) {

	magentawhite := color.New(color.FgMagenta).Add(color.BgWhite)
	magentawhite.Println("(1) Start Add Callback")

	//color.Cyan(fmt.Sprintf("SESSION : %v", sessionInfo))

	_company, _ := strconv.Atoi(sessionInfo["CompanyId"])
	_tenant, _ := strconv.Atoi(sessionInfo["TenantId"])
	_tryCount, _ := strconv.Atoi(sessionInfo["TryCount"])

	contactsList := []Contact{}
	_ = json.Unmarshal([]byte(sessionInfo["Contacts"]), &contactsList)

	isAllowCallback := GetAllowCallback(_company, _tenant, sessionInfo["CampaignId"])

	isdisconnectReasonAllowed, hangupGruop := ValidateDisconnectReason(sessionInfo["Reason"])

	magentawhite.Println(fmt.Sprintf("(2) Allow Callbacks : %t , Disconnect Reason Allowed : %t , Hangup Group : %s", isAllowCallback, isdisconnectReasonAllowed, hangupGruop))

	if isAllowCallback && isdisconnectReasonAllowed {
		magentawhite.Println("(3) CB Allowed")
		maxCallbackCount, callbackInterval, isReasonExists := GetCallbackDetails(_company, _tenant, sessionInfo["CampaignId"], hangupGruop)

		if isReasonExists {
			magentawhite.Println(fmt.Sprintf("(4) Reason Exists | maxCallbackCount : %d | Number : %s | TryCount : %d", maxCallbackCount, sessionInfo["Number"], _tryCount))
			if maxCallbackCount > 0 && sessionInfo["Number"] != "" && _tryCount > 0 && _tryCount <= maxCallbackCount {
				magentawhite.Println("(5) Try count valid")
				camIdInt, _ := strconv.Atoi(sessionInfo["CampaignId"])
				scheduleIdInt, _ := strconv.Atoi(sessionInfo["ScheduleId"])

				campaignInfo, isCamExists := GetCampaign(_company, _tenant, camIdInt)
				fmt.Println("isCamExists:: ", isCamExists)

				if isCamExists {

					magentawhite.Println("(6) Campaign Exist")

					scheduleInfo := CampaignShedule{}
					defaultScheduleInfo := CampaignShedule{}
					for _, schedule := range campaignInfo.CampScheduleInfo {
						if schedule.ScheduleId == scheduleIdInt {
							scheduleInfo = schedule
							break
						}
					}

					if scheduleInfo != defaultScheduleInfo {

						magentawhite.Println("(7) Schedule Exist")

						location, _ := time.LoadLocation(scheduleInfo.TimeZone)

						tmNow := time.Now().In(location)
						secCount := tmNow.Second() + callbackInterval
						callbackTime := time.Date(tmNow.Year(), tmNow.Month(), tmNow.Day(), tmNow.Hour(), tmNow.Minute(), secCount, 0, location)
						fmt.Println("callbackTime:: ", callbackTime)

						//tempCampaignEndDate, _ := time.Parse(layout2, campaignInfo.CampConfigurations.EndDate)
						//campaignEndDate := time.Date(tempCampaignEndDate.Year(), tempCampaignEndDate.Month(), tempCampaignEndDate.Day(), tempCampaignEndDate.Hour(), tempCampaignEndDate.Minute(), tempCampaignEndDate.Second(), 0, location)

						scheduleEndDate := scheduleInfo.EndDate

						fmt.Println("Callback:scheduleEndDate:: ", scheduleEndDate)
						if scheduleEndDate.After(callbackTime) {
							magentawhite.Println("(8) Callback ready to start")
							fmt.Println("Start to build CallbackInfo")
							//scheduleIdStr := strconv.Itoa(scheduleInfo.ScheduleId)
							validateAppoinment := CheckAppoinmentForCallback(_company, _tenant, sessionInfo["ScheduleId"], callbackTime, scheduleInfo.TimeZone)
							fmt.Println("validateAppoinmentFor Callback:: ", validateAppoinment)
							if validateAppoinment {
								magentawhite.Println("(9) Appointment valid")
								callbackObj := CampaignCallbackObj{}
								callbackObj.CampaignId = sessionInfo["CampaignId"]
								callbackObj.CallbackClass = "DIALER"
								callbackObj.CallbackType = "CALLBACK"
								callbackObj.CallbackCategory = "INTERNAL"
								callbackObj.CallBackCount = sessionInfo["TryCount"]
								callbackObj.ContactId = sessionInfo["Number"]
								callbackObj.DialoutTime = callbackTime
								callbackObj.PreviewData = sessionInfo["PreviewData"]
								callbackObj.ThirdPartyReference = sessionInfo["ThirdPartyReference"]

								callbackObj.OtherContacts = contactsList

								dialerAPIUrl := fmt.Sprintf("http://%s", CreateHost(lbIpAddress, lbPort))
								path := fmt.Sprintf("DVP/DialerAPI/ResumeCallback")

								u, _ := url.Parse(dialerAPIUrl)
								u.Path += path

								fmt.Println(u.String())
								cbUrl := u.String()

								jsonData, _ := json.Marshal(callbackObj)
								magentawhite.Println("(10) Uploading callback")

								go UploadCallbackInfo(_company, _tenant, callbackTime, sessionInfo["CampaignId"], "DIALER", "CALLBACK", "INTERNAL", cbUrl, string(jsonData))
							}
						}
					} else {
						magentawhite.Println("(11) Add Callback Failed, No Valied Schedule Found")
						fmt.Println("Add Callback Failed, No Valied Schedule Found")
					}

				} else {
					magentawhite.Println("(12) Add Callback Failed, No Existing Campaign Found")
					fmt.Println("Add Callback Failed, No Existing Campaign Found")
				}
				//RELEASING AGENT
				magentawhite.Println("(13) Releasing Agent")
				SendAgentIntegrationData(sessionInfo)
				SetAgentStatusArds(sessionInfo["CompanyId"], sessionInfo["TenantId"], sessionInfo["ArdsCategory"], sessionInfo["ResourceId"], sessionInfo["OriginalUuidARDS"], "Completed", sessionInfo["ARDSServerType"], sessionInfo["ARDSRequestType"])
				DecrConcurrentChannelCount(sessionInfo["ServerId"], sessionInfo["CampaignId"])
			} else {
				//SET NEXT CONTACT AND DIAL
				magentawhite.Println("(14) Set Next Contact")
				SetNextContact(contactsList, sessionInfo)
			}
		} else {
			//SET NEXT CONTACT AND DIAL
			magentawhite.Println("(15) Set Next Contact")
			SetNextContact(contactsList, sessionInfo)
		}

	} else {
		//SET NEXT CONTACT AND DIAL
		magentawhite.Println("(16) Set Next Contact")
		SetNextContact(contactsList, sessionInfo)

	}

}

func AddPhoneNumberToCallback(company, tenant, tryCount, campaignId, scheduleId, phoneNumber, disConnectkReason, ardsCategory, resourceId, sessionId, ardsServerType, ardsReqType, switchName string, sessionInfo map[string]string) {
	fmt.Println("start AddPhoneNumberToCallback")
	_company, _ := strconv.Atoi(company)
	_tenant, _ := strconv.Atoi(tenant)
	_tryCount, _ := strconv.Atoi(tryCount)
	isAllowCallback := GetAllowCallback(_company, _tenant, campaignId)
	fmt.Println("isAllowCallback:: ", isAllowCallback)
	isdisconnectReasonAllowed, hangupGruop := ValidateDisconnectReason(disConnectkReason)
	fmt.Println("isdisconnectReasonAllowed:: ", isdisconnectReasonAllowed)
	fmt.Println("hangupGruop:: ", hangupGruop)
	if isAllowCallback && isdisconnectReasonAllowed {
		maxCallbackCount, callbackInterval, isReasonExists := GetCallbackDetails(_company, _tenant, campaignId, hangupGruop)
		fmt.Println("maxCallbackCount:: ", maxCallbackCount)
		fmt.Println("callbackInterval:: ", callbackInterval)
		fmt.Println("isReasonExists:: ", isReasonExists)
		if isReasonExists {
			if maxCallbackCount > 0 && phoneNumber != "" && _tryCount > 0 && _tryCount <= maxCallbackCount {
				camIdInt, _ := strconv.Atoi(campaignId)
				scheduleIdInt, _ := strconv.Atoi(scheduleId)

				campaignInfo, isCamExists := GetCampaign(_company, _tenant, camIdInt)
				fmt.Println("isCamExists:: ", isCamExists)

				if isCamExists {

					scheduleInfo := CampaignShedule{}
					defaultScheduleInfo := CampaignShedule{}
					for _, schedule := range campaignInfo.CampScheduleInfo {
						if schedule.ScheduleId == scheduleIdInt {
							scheduleInfo = schedule
							break
						}
					}

					if scheduleInfo != defaultScheduleInfo {

						location, _ := time.LoadLocation(scheduleInfo.TimeZone)

						tmNow := time.Now().In(location)
						secCount := tmNow.Second() + callbackInterval
						callbackTime := time.Date(tmNow.Year(), tmNow.Month(), tmNow.Day(), tmNow.Hour(), tmNow.Minute(), secCount, 0, location)
						fmt.Println("callbackTime:: ", callbackTime)

						//tempCampaignEndDate, _ := time.Parse(layout2, campaignInfo.CampConfigurations.EndDate)
						//campaignEndDate := time.Date(tempCampaignEndDate.Year(), tempCampaignEndDate.Month(), tempCampaignEndDate.Day(), tempCampaignEndDate.Hour(), tempCampaignEndDate.Minute(), tempCampaignEndDate.Second(), 0, location)

						scheduleEndDate := scheduleInfo.EndDate

						fmt.Println("Callback:scheduleEndDate:: ", scheduleEndDate)
						if scheduleEndDate.After(callbackTime) {
							fmt.Println("Start to build CallbackInfo")
							//scheduleIdStr := strconv.Itoa(scheduleInfo.ScheduleId)
							validateAppoinment := CheckAppoinmentForCallback(_company, _tenant, scheduleId, callbackTime, scheduleInfo.TimeZone)
							fmt.Println("validateAppoinmentFor Callback:: ", validateAppoinment)
							if validateAppoinment {
								callbackObj := CampaignCallbackObj{}
								callbackObj.CampaignId = campaignId
								callbackObj.CallbackClass = "DIALER"
								callbackObj.CallbackType = "CALLBACK"
								callbackObj.CallbackCategory = "INTERNAL"
								callbackObj.CallBackCount = tryCount
								callbackObj.ContactId = phoneNumber
								callbackObj.DialoutTime = callbackTime
								callbackObj.PreviewData = sessionInfo["PreviewData"]
								callbackObj.BusinessUnit = sessionInfo["BusinessUnit"]

								dialerAPIUrl := fmt.Sprintf("http://%s", CreateHost(lbIpAddress, lbPort))
								path := fmt.Sprintf("DVP/DialerAPI/ResumeCallback")

								u, _ := url.Parse(dialerAPIUrl)
								u.Path += path

								fmt.Println(u.String())
								cbUrl := u.String()

								jsonData, _ := json.Marshal(callbackObj)
								go UploadCallbackInfo(_company, _tenant, callbackTime, campaignId, "DIALER", "CALLBACK", "INTERNAL", cbUrl, string(jsonData))
							}
						}
					} else {
						fmt.Println("Add Callback Failed, No Valied Schedule Found")
					}

				} else {
					fmt.Println("Add Callback Failed, No Existing Campaign Found")
				}
			}
		}
	}

	SendAgentIntegrationData(sessionInfo)
	SetAgentStatusArds(company, tenant, ardsCategory, resourceId, sessionId, "Completed", ardsServerType, ardsReqType)
	DecrConcurrentChannelCount(switchName, campaignId)
}

func ResumeCampaignCallback(company, tenant, callbackCount, campaignId int, number string, otherContacts []Contact, previewData, businessUnit, thirdpartyRef string, skills []string) {
	blackgreen := color.New(color.FgBlack).Add(color.BgGreen)
	campaignIdStr := strconv.Itoa(campaignId)
	_tryCount := callbackCount + 1
	campaign, isCampaignExists := GetCampaign(company, tenant, campaignId)
	if isCampaignExists {
		blackgreen.Println(fmt.Sprintf("CALLBACK CAMPAIGN %d EXIST", campaignId))
		camScheduleStr := strconv.Itoa(campaign.CampScheduleInfo[0].CamScheduleId)
		//numberWithTryCount := fmt.Sprintf("%s:%d:%s", number, _tryCount, previewData)
		if campaign.CampConfigurations.NumberLoadingMethod == "CONTACT" {
			contactDet := ContactsDetails{Phone: number, Api_Contacts: otherContacts, PreviewData: previewData, TryCount: _tryCount, Skills: skills, Thirdpartyreference: thirdpartyRef}
			blackgreen.Println(fmt.Sprintf("Adding contacts to front : %v", contactDet))
			AddContactToFront(company, tenant, campaignIdStr, contactDet)
		} else {
			numberInfo := CampaignDialNumber{}
			numberInfo.PhoneNumber = number
			numberInfo.TryCount = strconv.Itoa(_tryCount)
			numberInfo.BusinessUnit = businessUnit
			numberInfo.PreviewData = previewData

			blackgreen.Println(fmt.Sprintf("Adding number to front - Number | %s", number))
			AddNumberToFront(company, tenant, campaignIdStr, camScheduleStr, numberInfo)
		}

	} else {
		blackgreen.Println(fmt.Sprintf("CALLBACK CAMPAIGN %d DOES NOT EXIST", campaignId))
	}
}
