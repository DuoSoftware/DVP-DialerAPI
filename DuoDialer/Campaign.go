package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	uuid "github.com/satori/go.uuid"
)

//----------Ongoing Campaign Count-----------------------
func AddOnGoingCampaignCount() {
	dialerCampaignCountKey := fmt.Sprintf("DialerOnGoingCampaignCount:%s:%s", lbIpAddress, dialerId)
	result := RedisAdd(dialerCampaignCountKey, "0")
	fmt.Println("Add DialerOnGoingCampaignCount to Redis: ", result)
}

func GetOnGoingCampaignCount() int {
	dialerCampaignCountKey := fmt.Sprintf("DialerOnGoingCampaignCount:%s:%s", lbIpAddress, dialerId)
	countStr := RedisGet(dialerCampaignCountKey)
	count, _ := strconv.Atoi(countStr)
	DialerLog(fmt.Sprintf("OnGoingCampaignCount: %s", countStr))
	return count
}

func IncrementOnGoingCampaignCount() {
	dialerCampaignCountKey := fmt.Sprintf("DialerOnGoingCampaignCount:%s:%s", lbIpAddress, dialerId)
	incValue := RedisIncr(dialerCampaignCountKey)
	DialerLog(fmt.Sprintf("IncrementedOnGoingCampaignCount: %d", incValue))
}

func DecrementOnGoingCampaignCount() {
	dialerCampaignCountKey := fmt.Sprintf("DialerOnGoingCampaignCount:%s:%s", lbIpAddress, dialerId)
	dncValue := RedisIncrBy(dialerCampaignCountKey, -1)
	DialerLog(fmt.Sprintf("DecrementOnGoingCampaignCount: %d", dncValue))
}

func DecrementOnGoingCampaignCountOther(oDialerId string) {
	dialerCampaignCountSKey := fmt.Sprintf("DialerOnGoingCampaignCount:*:%s", oDialerId)
	searchdialer := RedisSearchKeys(dialerCampaignCountSKey)
	if len(searchdialer) > 0 {
		dncValue := RedisIncrBy(searchdialer[0], -1)
		DialerLog(fmt.Sprintf("DecrementOnGoingCampaignCountOther: %d", dncValue))
	}
}

//----------Campaign Status-----------------------
func SetCampaignStatus(campaignId, status string, company, tenant int) {
	campaignStatusKey := fmt.Sprintf("CampaignStatus:%s:%d:%d:%s", dialerId, company, tenant, campaignId)
	result := RedisSet(campaignStatusKey, status)
	DialerLog(fmt.Sprintf("SetCampaignStatus CampaignId: %s  Status: %s Result: %s", campaignStatusKey, status, result))
}

func GetCampaignStatus(campaignId string, company, tenant int) string {
	campaignStatusKey := fmt.Sprintf("CampaignStatus:%s:%d:%d:%s", dialerId, company, tenant, campaignId)
	result := RedisGet(campaignStatusKey)
	DialerLog(fmt.Sprintf("GetCampaignStatus CampaignId: %s Result: %s", campaignStatusKey, result))
	return result
}

func RemoveCampaignStatus(campaignId string, company, tenant int) {
	campaignStatusKey := fmt.Sprintf("CampaignStatus:%s:%d:%d:%s", dialerId, company, tenant, campaignId)
	result := RedisRemove(campaignStatusKey)
	DialerLog(fmt.Sprintf("GetCampaignStatus CampaignId: %s  Result: %t", campaignStatusKey, result))
}

func RemoveCampaignStatusOther(oDialerId, campaignId string, company, tenant int) {
	campaignStatusKey := fmt.Sprintf("CampaignStatus:%s:%d:%d:%s", oDialerId, company, tenant, campaignId)
	result := RedisRemove(campaignStatusKey)
	DialerLog(fmt.Sprintf("RemoveCampaignStatusOther CampaignId: %s Result: %t", campaignStatusKey, result))
}

//----------Campaign and schedule status-----------------
func SetScheduleStatus(campaignId, scheduleId, status string, company, tenant int) {
	scheduleStatusKey := fmt.Sprintf("ScheduleStatus:%s:%d:%d:%s:%s", dialerId, company, tenant, campaignId, scheduleId)
	result := RedisSet(scheduleStatusKey, status)
	fmt.Println("SetScheduleStatus CampaignId: %s", scheduleStatusKey, " Result: %s", result)
}

func GetScheduleStatus(campaignId, scheduleId string, company, tenant int) string {
	scheduleStatusKey := fmt.Sprintf("ScheduleStatus:%s:%d:%d:%s:%s", dialerId, company, tenant, campaignId, scheduleId)
	result := RedisGet(scheduleStatusKey)
	fmt.Println("GetScheduleStatus CampaignId: %s", scheduleStatusKey, " Result: %s", result)
	return result
}

func RemoveScheduleStatus(campaignId, scheduleId string, company, tenant int) {
	scheduleStatusKey := fmt.Sprintf("ScheduleStatus:%s:%d:%d:%s:%s", dialerId, company, tenant, campaignId, scheduleId)
	result := RedisRemove(scheduleStatusKey)
	fmt.Println("GetScheduleStatus CampaignId: %s", scheduleStatusKey, " Result: %s", result)
}

func RemoveScheduleStatusOther(oDialerId, campaignId, scheduleId string, company, tenant int) {
	scheduleStatusKey := fmt.Sprintf("ScheduleStatus:%s:%d:%d:%s:%s", oDialerId, company, tenant, campaignId, scheduleId)
	result := RedisRemove(scheduleStatusKey)
	fmt.Println("RemoveScheduleStatusOther CampaignId: %s", scheduleStatusKey, " Result: %s", result)
}

//----------Campaign-----------------------
func AddCampaignToDialer(campaignD Campaign) {
	if len(campaignD.CampScheduleInfo) > 0 {
		campaignKey := fmt.Sprintf("Campaign:%s:%d:%d:%d", dialerId, campaignD.CompanyId, campaignD.TenantId, campaignD.CampaignId)
		searchCamp := fmt.Sprintf("Campaign:*:%d:%d:%d", campaignD.CompanyId, campaignD.TenantId, campaignD.CampaignId)
		existingKeys := RedisSearchKeys(searchCamp)

		companyToken := fmt.Sprintf("%d:%d", campaignD.TenantId, campaignD.CompanyId)

		defaultScheduleId := strconv.Itoa(campaignD.CampScheduleInfo[0].ScheduleId)
		defaultStartDate, defaultEndDate, defaultTimeZone := GetTimeZoneFroSchedule(companyToken, defaultScheduleId)

		location, _ := time.LoadLocation(defaultTimeZone)
		defaultCampaignStartDate, _ := time.Parse(layout2, defaultStartDate)
		defaultCampaignEndDate, _ := time.Parse(layout2, defaultEndDate)

		tempCampaignStartDate := time.Date(defaultCampaignStartDate.Year(), defaultCampaignStartDate.Month(), defaultCampaignStartDate.Day(), 0, 0, 0, 0, location)
		tempCampaignEndDate := time.Date(defaultCampaignEndDate.Year(), defaultCampaignEndDate.Month(), defaultCampaignEndDate.Day(), 0, 0, 0, 0, location)

		campaignD.CampConfigurations.StartDate = tempCampaignStartDate
		campaignD.CampConfigurations.EndDate = tempCampaignEndDate
		campaignD.CampConfigurations.StartTimeZone = defaultTimeZone
		campaignD.CampConfigurations.EndTimeZone = defaultTimeZone

		for i, campSchedule := range campaignD.CampScheduleInfo {
			scheduleId := strconv.Itoa(campSchedule.ScheduleId)
			startDate, endDate, timeZone := GetTimeZoneFroSchedule(companyToken, scheduleId)

			scheduleLocation, _ := time.LoadLocation(timeZone)
			scheduleStartDate, _ := time.Parse(layout2, startDate)
			scheduleEndDate, _ := time.Parse(layout2, endDate)

			tempScheduleStartDate := time.Date(scheduleStartDate.Year(), scheduleStartDate.Month(), scheduleStartDate.Day(), 0, 0, 0, 0, scheduleLocation)
			tempScheduleEndDate := time.Date(scheduleEndDate.Year(), scheduleEndDate.Month(), scheduleEndDate.Day(), 0, 0, 0, 0, scheduleLocation)

			campaignD.CampScheduleInfo[i].StartDate = tempScheduleStartDate
			campaignD.CampScheduleInfo[i].EndDate = tempScheduleEndDate
			campaignD.CampScheduleInfo[i].TimeZone = timeZone

			DialerLog(fmt.Sprintf("Add Schedule Time Zone::%s", timeZone))
			DialerLog(fmt.Sprintf("Add Schedule Start Time::%s", tempScheduleStartDate.String()))
			DialerLog(fmt.Sprintf("Add Schedule End Time::%s", tempScheduleEndDate.String()))

			if tempScheduleStartDate.Before(tempCampaignStartDate) {
				campaignD.CampConfigurations.StartDate = tempScheduleStartDate
				campaignD.CampConfigurations.StartTimeZone = timeZone
			}

			if tempScheduleEndDate.After(tempCampaignEndDate) {
				campaignD.CampConfigurations.EndDate = tempScheduleEndDate
				campaignD.CampConfigurations.EndTimeZone = timeZone
			}
		}

		if len(existingKeys) == 0 {
			campaignJson, _ := json.Marshal(campaignD)
			result := RedisAdd(campaignKey, string(campaignJson))
			DialerLog(fmt.Sprintf("Add Campaign to Redis: %s Result: %s", campaignKey, result))
			if result == "OK" {
				campIdStr := strconv.Itoa(campaignD.CampaignId)
				channelCountStr := strconv.Itoa(campaignD.CampConfigurations.ChannelConcurrency)
				//SetCampaignTimeZone(campIdStr, campaignD.CompanyId, campaignD.TenantId, timeZone)
				IncrementOnGoingCampaignCount()
				SetCampChannelMaxLimitDirect(campIdStr, channelCountStr)
				AddCampaignCallbackConfigInfo(campaignD.CompanyId, campaignD.TenantId, campaignD.CampaignId, campaignD.CampConfigurations.ConfigureId)
				color.Green("STARTING NEW CAMPAIGN - FROM THIS DIALER - SET STATUS TO START")
				SetCampaignStatus(campIdStr, "Start", campaignD.CompanyId, campaignD.TenantId)
				UpdateCampaignStartStatus(campaignD.CompanyId, campaignD.TenantId, campIdStr)
				UpdateCampaignStartAndEndDate(campaignD.CompanyId, campaignD.TenantId, campaignD.CampaignId, campaignD.CampConfigurations.ConfigureId, campaignD.CampConfigurations.StartDate.Format("02 Jan 06 15:04 -0700"), campaignD.CampConfigurations.EndDate.Format("02 Jan 06 15:04 -0700"))
			}
		} else {
			splitVals := strings.Split(existingKeys[0], ":")
			preDialerId := splitVals[1]
			campIdStr := strconv.Itoa(campaignD.CampaignId)
			color.Yellow(fmt.Sprintf("Removing campaign:%d From Prev Dialer :%s and assining to current Dialer :%s", campaignD.CampaignId, preDialerId, dialerId))
			RemoveCampaignFromOtherDialer(preDialerId, campIdStr, campaignD.CompanyId, campaignD.TenantId)

			campaignJson, _ := json.Marshal(campaignD)
			result := RedisAdd(campaignKey, string(campaignJson))
			DialerLog(fmt.Sprintf("Add Campaign to Redis: %s  Result: %s", campaignKey, result))
			if result == "OK" {
				//SetCampaignTimeZone(campIdStr, campaignD.CompanyId, campaignD.TenantId, timeZone)
				IncrementOnGoingCampaignCount()
				SetCampaignStatus(campIdStr, "Resume", campaignD.CompanyId, campaignD.TenantId)
				UpdateCampaignStartStatus(campaignD.CompanyId, campaignD.TenantId, campIdStr)
			}
		}
		//Add Campaign Record
		AddCampaignDataRealtime(campaignD)
	} else {
		color.Red(fmt.Sprintf("Add Campaign : %s to Redis failed No schedule found", campaignD.CampaignName))
	}
}

func GetCampaign(company, tenant, campaignId int) (Campaign, bool) {
	campaignKey := fmt.Sprintf("Campaign:%s:%d:%d:%d", dialerId, company, tenant, campaignId)
	isExists := RedisCheckKeyExist(campaignKey)
	if isExists {
		campJson := RedisGet(campaignKey)
		var camp Campaign
		json.Unmarshal([]byte(campJson), &camp)
		return camp, isExists
	} else {
		return Campaign{}, isExists
	}
}

func GetAllRunningCampaign() []Campaign {
	searchKey := fmt.Sprintf("Campaign:%s:*", dialerId)
	allCampaignKeys := RedisSearchKeys(searchKey)

	allCampaigns := make([]Campaign, 0)
	for _, key := range allCampaignKeys {
		campJson := RedisGet(key)
		var camp Campaign
		json.Unmarshal([]byte(campJson), &camp)
		allCampaigns = append(allCampaigns, camp)
	}
	DialerLog(fmt.Sprintf("GetAllRunningCampaign:: %+v", allCampaigns))
	return allCampaigns
}

func RemoveCampaignFromDialer(campaignId string, company, tenant int) {
	campaignKey := fmt.Sprintf("Campaign:%s:%d:%d:%s", dialerId, company, tenant, campaignId)
	result := RedisRemove(campaignKey)
	DialerLog(fmt.Sprintf("Remove Campaign from Redis: %s  Result: %t", campaignKey, result))
	if result == true {
		DecrementOnGoingCampaignCount()
		RemoveCampaignStatus(campaignId, company, tenant)
		RemoveNumbers(company, tenant, campaignId)
		RemoveNumberStatusKey(company, tenant, campaignId)
		RemoveCampChannelMaxLimit(campaignId)
		RemoveCampaignConnectedCount(company, tenant, campaignId)
		RemoveCampaignDialCount(company, tenant, campaignId)
		RemoveCampConcurrentChannelCount(campaignId)
		RemoveCampaignCallbackConfigInfo(company, tenant, campaignId)
	}
}

func RemoveCampaignFromOtherDialer(oDialerId, campaignId string, company, tenant int) {
	campaignKey := fmt.Sprintf("Campaign:%s:%d:%d:%s", oDialerId, company, tenant, campaignId)
	result := RedisRemove(campaignKey)
	DialerLog(fmt.Sprintf("Remove Campaign from Redis: %s Result: %s", campaignKey, result))
	if result == true {
		DecrementOnGoingCampaignCountOther(oDialerId)
		RemoveCampaignStatusOther(oDialerId, campaignId, company, tenant)
	}
}

//func SetCampaignTimeZone(campaignId string, company, tenant int, timeZone string) {
//	campaignKey := fmt.Sprintf("Campaign:%s:%d:%d:%s", dialerId, company, tenant, campaignId)

//	campJson := RedisGet(campaignKey)
//	var campObj Campaign
//	json.Unmarshal([]byte(campJson), &campObj)

//	campObj.TimeZone = timeZone

//	campaignJson, _ := json.Marshal(campObj)

//	result := RedisAdd(campaignKey, string(campaignJson))
//	fmt.Println("Add Campaign to Redis: ", campaignKey, " Result: ", result)
//	if result == "OK" {
//		fmt.Println("Update Campaign TimeZone success")
//	}
//}

//----------Campaign Manager Service-----------------------
func RequestCampaign(requestCount int) []Campaign {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in RequestCampaign", r)
		}
	}()
	//Request campaign from Campaign Manager service
	campaignDetails := make([]Campaign, 0)

	client := &http.Client{}

	jwtToken := fmt.Sprintf("Bearer %s", accessToken)
	request := fmt.Sprintf("http://%s/DVP/API/1.0.0.0/CampaignManager/Campaigns/State/Pending/%d", CreateHost(campaignServiceHost, campaignServicePort), requestCount)
	DialerLog(fmt.Sprintf("Start RequestCampaign request: %s", request))
	req, _ := http.NewRequest("GET", request, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("authorization", jwtToken)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return campaignDetails
	}
	defer resp.Body.Close()

	response, _ := ioutil.ReadAll(resp.Body)

	DialerLog(fmt.Sprintf("Campaign Data:: %s", string(response)))

	var campaignResult CampaignResult
	json.Unmarshal(response, &campaignResult)
	if campaignResult.IsSuccess == true {
		for _, camRes := range campaignResult.Result {
			color.Blue(fmt.Sprintf("Campaign FOUND FOR DIALING : %+v", camRes))
			campaignDetails = append(campaignDetails, camRes)
		}
	}

	DialerLog(fmt.Sprintf("campaignDetails:: %+v", campaignDetails))
	return campaignDetails
}

func UpdateCampaignStatus(company, tenant int, campaignId string) (actualState string) {
	defer func() {
		if r := recover(); r != nil {
			color.Red(fmt.Sprintf("Recovered in UpdateCampaignStatus %+v", r))
		}
	}()
	//Send CampaignStatus to Campaign Manager
	jwtToken := fmt.Sprintf("Bearer %s", accessToken)
	internalAuthToken := fmt.Sprintf("%d:%d", tenant, company)
	DialerLog(fmt.Sprintf("Start UpdateCampaignStatus Auth: %s CampaignId: %s DialerId: %s", internalAuthToken, campaignId, dialerId))
	client := &http.Client{}

	currentState := GetCampaignStatus(campaignId, company, tenant)
	actualState = currentState

	request := fmt.Sprintf("http://%s/DVP/API/1.0.0.0/CampaignManager/Campaign/%s/Operations/State/%s/%s", CreateHost(campaignServiceHost, campaignServicePort), campaignId, dialerId, currentState)
	DialerLog(fmt.Sprintf("Start UpdateCampaignStatus request: %s", request))
	req, _ := http.NewRequest("GET", request, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", jwtToken)
	req.Header.Set("companyinfo", internalAuthToken)
	resp, err := client.Do(req)
	if err != nil {
		color.Red(err.Error())
	}
	defer resp.Body.Close()

	response, _ := ioutil.ReadAll(resp.Body)
	var campaignStatusResult CampaignStatusResult
	json.Unmarshal(response, &campaignStatusResult)
	if campaignStatusResult.IsSuccess == true {
		emtResult := CampaignState{}
		if campaignStatusResult.Result != emtResult {
			state := campaignStatusResult.Result.CampaignState
			camId := campaignStatusResult.Result.CampaignId
			dId := campaignStatusResult.Result.DialerId

			campIdStr := strconv.Itoa(camId)

			currentStateLower := strings.ToLower(currentState)
			stateLower := strings.ToLower(state)

			if campaignId == campIdStr && dialerId == dId && currentStateLower != stateLower {
				switch state {
				case "stop":
					SetCampaignStatus(campIdStr, "Stop", company, tenant)
					actualState = "Stop"
					break
				case "pause":
					SetCampaignStatus(campIdStr, "Pause", company, tenant)
					actualState = "Pause"
					break
				case "resume":
					SetCampaignStatus(campIdStr, "Resume", company, tenant)
					actualState = "Resume"
					break
				case "end":
					SetCampaignStatus(campIdStr, "End", company, tenant)
					actualState = "End"
					break
				default:
					break
				}
			}
		}
	}

	return
}

func UpdateCampaignStartStatus(company, tenant int, campaignId string) {
	defer func() {
		if r := recover(); r != nil {
			color.Red(fmt.Sprintf("Recovered in UpdateCampaignStartStatus %+v", r))
		}
	}()
	//Send CampaignStatus to Campaign Manager
	state := CampaignStart{}
	camIdInt, _ := strconv.Atoi(campaignId)
	state.CampaignId = camIdInt
	state.DialerId = dialerId

	jsonData, _ := json.Marshal(state)

	jwtToken := fmt.Sprintf("Bearer %s", accessToken)
	internalAuthToken := fmt.Sprintf("%d:%d", tenant, company)
	serviceurl := fmt.Sprintf("http://%s/DVP/API/1.0.0.0/CampaignManager/Campaign/%s/Operations/%s", CreateHost(campaignServiceHost, campaignServicePort), campaignId, dialerId)
	req, err := http.NewRequest("POST", serviceurl, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", jwtToken)
	req.Header.Set("companyinfo", internalAuthToken)
	DialerLog(fmt.Sprintf("request:%s", serviceurl))
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
}

func UpdateCampaignStartAndEndDate(company, tenant, campaignId, configId int, startDate, endDate string) {
	defer func() {
		if r := recover(); r != nil {
			color.Red(fmt.Sprintf("Recovered in UpdateCampaignStartAndEndDate %v", r))
		}
	}()

	//fmt.Println(company, " :: ", tenant, " :: ", campaignId, " :: ", configId, " :: ", startDate, " :: ", endDate)

	jwtToken := fmt.Sprintf("Bearer %s", accessToken)
	internalAuthToken := fmt.Sprintf("%d:%d", tenant, company)
	serviceurl := fmt.Sprintf("http://%s/DVP/API/1.0.0.0/CampaignManager/Campaign/%d/Configuration/%d/StartDate/%s/EndDate/%s", CreateHost(campaignServiceHost, campaignServicePort), campaignId, configId, startDate, endDate)

	req, err := http.NewRequest("PUT", serviceurl, bytes.NewBuffer([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", jwtToken)
	req.Header.Set("companyinfo", internalAuthToken)
	DialerLog(fmt.Sprintf("request:%s", serviceurl))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		color.Red(err.Error())
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
}

//----------Campaign Channel Max Limit-----------------------
func IncrCampChannelMaxLimit(campaignId string) {
	cmcl := fmt.Sprintf("CampaignMaxCallLimit:%s", campaignId)
	RedisIncr(cmcl)
}

func DecrCampChannelMaxLimit(campaignId string) {
	cmcl := fmt.Sprintf("CampaignMaxCallLimit:%s", campaignId)
	decValue := RedisIncrBy(cmcl, -1)
	if decValue < 0 {
		RedisSet(cmcl, "0")
	}
}

func SetCampChannelMaxLimit(campaignId string) {
	ids := strings.Split(campaignId, "_")
	if len(ids) == 2 {
		cmcl := fmt.Sprintf("CampaignMaxCallLimit:%s", ids[0])
		RedisSet(cmcl, ids[1])
	}
}

func SetCampChannelMaxLimitDirect(campaignId, channelcount string) {
	cmcl := fmt.Sprintf("CampaignMaxCallLimit:%s", campaignId)
	RedisSet(cmcl, channelcount)
}

func GetCampMaxChannelLimit(campaignId string) int {
	cmcl := fmt.Sprintf("CampaignMaxCallLimit:%s", campaignId)
	maxChannelCount := RedisGet(cmcl)
	if maxChannelCount == "" {
		RedisSet(cmcl, "0")
	}
	value, err := strconv.Atoi(maxChannelCount)
	if err != nil {
		fmt.Println(err.Error())
		return 0
	} else {
		return value
	}
}

func RemoveCampChannelMaxLimit(campaignId string) {
	cmcl := fmt.Sprintf("CampaignMaxCallLimit:%s", campaignId)
	RedisRemove(cmcl)
}

//----------Campaign Dial Count-----------------------
func IncrCampaignDialCount(company, tenant int, campaignId string) {
	cmcl := fmt.Sprintf("CampaignDialCount:%d:%d:%s", company, tenant, campaignId)
	RedisIncr(cmcl)
}

func GetCampaignDialCount(company, tenant int, campaignId string) int {
	cmcl := fmt.Sprintf("CampaignDialCount:%d:%d:%s", company, tenant, campaignId)
	value := RedisGet(cmcl)
	count, _ := strconv.Atoi(value)
	return count
}

func RemoveCampaignDialCount(company, tenant int, campaignId string) {
	cmcl := fmt.Sprintf("CampaignDialCount:%d:%d:%s", company, tenant, campaignId)
	RedisRemove(cmcl)
}

//----------Campaign Connected Count-----------------------
func IncrCampaignConnectedCount(company, tenant int, campaignId string) {
	cmcl := fmt.Sprintf("CampaignConnectedCount:%d:%d:%s", company, tenant, campaignId)
	RedisIncr(cmcl)
}

func GetCampaignConnectedCount(company, tenant int, campaignId string) int {
	cmcl := fmt.Sprintf("CampaignConnectedCount:%d:%d:%s", company, tenant, campaignId)
	value := RedisGet(cmcl)
	count, _ := strconv.Atoi(value)
	return count
}

func RemoveCampaignConnectedCount(company, tenant int, campaignId string) {
	cmcl := fmt.Sprintf("CampaignConnectedCount:%d:%d:%s", company, tenant, campaignId)
	RedisRemove(cmcl)
}

//----------Run Campaign-----------------------
func StartCampaign(campaignId, campaignName, dialoutMec, CampaignChannel, camClass, camType, camCategory, scheduleId, camScheduleId, resourceServerId, extention, defaultAni string, company, tenant, campaignMaxChannelCount int, integrationData *IntegrationConfig, numLoadingMethod string, duplicateNumTimeout int) {
	color.Yellow(fmt.Sprintf("=======CampaignName : %s - %v =======", campaignName, *integrationData))
	campStatus := GetCampaignStatus(campaignId, company, tenant)
	SetCampaignStatus(campaignId, "Running", company, tenant)
	emtAppoinment := Appoinment{}
	defResourceServerInfo := ResourceServerInfo{}
	internalAuthToken := fmt.Sprintf("%d:%d", tenant, company)

	appment, timeZone, appmntEndTime := CheckAppoinmentForCampaign(internalAuthToken, scheduleId)

	location, _ := time.LoadLocation(timeZone)

	resourceServerInfos := GetResourceServerInfo(company, tenant, resourceServerId, CampaignChannel)

	if appment != emtAppoinment && resourceServerInfos != defResourceServerInfo {
		color.Green(fmt.Sprintf("APPOINTMENT FOUND FOR CAMPAIGN : %s", campaignId))
		/* campStatus := GetCampaignStatus(campaignId, company, tenant)

		SetCampaignStatus(campaignId, "Running", company, tenant) */
		maxChannelLimitStr := strconv.Itoa(campaignMaxChannelCount)
		SetCampChannelMaxLimitDirect(campaignId, maxChannelLimitStr)

		numLoadingStatusKey := fmt.Sprintf("PhoneNumberLoading:%d:%d:%s:%s", company, tenant, campaignId, camScheduleId)

		if campStatus == "Start" || (campStatus == "Waiting for Appoinment" && !RedisCheckKeyExist(numLoadingStatusKey)) {
			color.Cyan("LOADING NUMBER SET")
			SetDncNumbersFromNumberBase(company, tenant)
			LoadInitialNumberSet(company, tenant, campaignId, camScheduleId, numLoadingMethod)
		}

		//endTime, _ := time.Parse(layout1, appment.EndTime)
		//timeNow := time.Now().UTC()
		//appmntEndTime := time.Date(timeNow.Year(), timeNow.Month(), timeNow.Day(), endTime.Hour(), endTime.Minute(), endTime.Second(), 0, time.UTC)

		maxChannelLimit := GetMaxChannelLimit(resourceServerInfos.ResourceServerId)
		maxCampaignChannelLimit := GetCampMaxChannelLimit(campaignId)
		if maxCampaignChannelLimit < 1 {
			maxCampaignChannelLimit = 10
		}

		campIdInt, _ := strconv.Atoi(campaignId)

		UpdateCampaignRealtimeField("OperationalStatus", "DIALING", tenant, company, campIdInt)

		for {
			campStatus = GetCampaignStatus(campaignId, company, tenant)
			if campStatus == "Running" {
				tm := time.Now().In(location)
				DialerLog(fmt.Sprintf("endTime: %s", appmntEndTime.String()))
				DialerLog(fmt.Sprintf("timeNW: %s", tm.String()))
				if tm.Before(appmntEndTime) {

					switch CampaignChannel {

					case "CALL":

						cchannelCountS, cchannelCountC := GetConcurrentChannelCount(resourceServerInfos.ResourceServerId, campaignId)

						DialerLog(fmt.Sprintf("resourceServerInfos.CallServerId: %s", resourceServerInfos.ResourceServerId))
						DialerLog(fmt.Sprintf("MaxCallServerChannelLimit: %d", maxChannelLimit))
						DialerLog(fmt.Sprintf("maxCampaignChannelLimit: %d", maxCampaignChannelLimit))
						DialerLog(fmt.Sprintf("ConcurrentResourceServerChannel: %v", cchannelCountS))
						DialerLog(fmt.Sprintf("ConcurrentCampaignChannel: %v", cchannelCountC))

						if cchannelCountS < maxChannelLimit && cchannelCountC < maxCampaignChannelLimit {

							number, tryCount, numExtraData, businessUnit, thirdpartyref, contacts, numberWiseSills := GetNumberToDial(company, tenant, campaignId, camScheduleId, numLoadingMethod)

							color.Green(fmt.Sprintf("Number:%s | TryCount:%s | numExtraData:%s | businessUnit:%s", number, tryCount, numExtraData, businessUnit))

							if number == "" {
								numberCount := GetNumberCount(company, tenant, campaignId, camScheduleId)
								if numberCount == 0 {
									//SetCampaignStatus(campaignId, "End", company, tenant)
									//RemoveCampaignFromDialer(campaignId, company, tenant)
									SetCampaignStatus(campaignId, "PauseByDialer", company, tenant)
									//SetCampChannelMaxLimitDirect(campaignId, "0")
									return
								}
							} else {

								tCount, _ := strconv.Atoi(tryCount)

								duplicatesNotFound := CheckDuplicates(company, tenant, campaignId, camScheduleId, number, duplicateNumTimeout, tCount)

								if duplicatesNotFound {

									if tCount <= 1 {
										PublishCampaignCallCounts("", "NUMBERADDED", strconv.Itoa(company), strconv.Itoa(tenant), campaignId, "")
									}

									uuid := GetUuid(resourceServerInfos.Url)
									context := fmt.Sprintf("%d_%d_dialer", tenant, company)
									trunkCode, ani, dnis, xGateway := GetTrunkCode(internalAuthToken, defaultAni, number, context, businessUnit)

									if trunkCode != "" && uuid != "" {
										switch dialoutMec {
										case "BLAST":
											color.Cyan(fmt.Sprintf("======= STARTING BLAST DIALER : %s =======", campaignId))
											go DialNumber(company, tenant, resourceServerInfos, campaignId, scheduleId, campaignName, uuid, ani, trunkCode, dnis, xGateway, tryCount, extention, integrationData, &contacts, thirdpartyref, businessUnit)
											break
										case "FIFO":
											color.Cyan(fmt.Sprintf("======= STARTING FIFO DIALER : %s =======", campaignId))
											go DialNumberFIFO(company, tenant, resourceServerInfos, campaignId, scheduleId, campaignName, uuid, ani, trunkCode, dnis, xGateway, extention, integrationData, &contacts)
											break
										case "PREVIEW":
											color.Cyan(fmt.Sprintf("======= STARTING PREVIEW DIALER : %s ======= %v", campaignId, contacts))
											go AddPreviewDialRequest(company, tenant, resourceServerInfos, campaignId, scheduleId, campaignName, dialoutMec, uuid, ani, trunkCode, dnis, xGateway, numExtraData, tryCount, extention, integrationData, &contacts, thirdpartyref, businessUnit, numberWiseSills)
											break
										case "AGENT":
											color.Cyan(fmt.Sprintf("======= STARTING AGENT DIALER : %s =======", campaignId))
											go AddAgentDialRequest(company, tenant, resourceServerInfos, campaignId, scheduleId, campaignName, dialoutMec, uuid, ani, trunkCode, dnis, xGateway, numExtraData, tryCount, extention, integrationData, &contacts, thirdpartyref, businessUnit, numberWiseSills)
											break
										}
									} else {
										color.Yellow("======= TRUNK OR UUID NOT FOUND =======")
									}
								}

								time.Sleep(100 * time.Millisecond)

							}

						} else {
							color.Cyan("CHANNEL COUNT EXCEEDED : " + campaignName)
							time.Sleep(500 * time.Millisecond)
						}
						break

					case "EMAIL":

						templates := RequestCampaignAddtionalData(company, tenant, campaignId, "BLAST", "EMAIL", "TEMPLATE")
						attachmentNames := RequestCampaignAddtionalData(company, tenant, campaignId, "BLAST", "EMAIL", "ATTACHMENT")

						email, _, numExtraData, _, _, _, _ := GetNumberToDial(company, tenant, campaignId, camScheduleId, "")
						if email == "" {
							emailCount := GetNumberCount(company, tenant, campaignId, camScheduleId)
							if emailCount == 0 {
								SetCampaignStatus(campaignId, "PauseByDialer", company, tenant)
								return
							}
						} else {
							uuidV4 := uuid.NewV4()
							sessionId := uuidV4.String()
							emailData := make(map[string]interface{})

							emailData["company"] = company
							emailData["tenant"] = tenant
							emailData["to"] = email
							emailData["from"] = defaultAni
							emailData["subject"] = campaignName

							InitiateSessionInfo(company, tenant, 240, "Campaign", "Email", "BlastDial", "1", campaignId, scheduleId, campaignName, sessionId, email, "start", "dial_start", time.Now().UTC().Format(layout4), resourceServerInfos.ResourceServerId, nil, nil, "", "", "")

							if len(templates) > 0 {
								emailData["template"] = templates[0]

								if numExtraData != "" {
									var params map[string]interface{}
									json.Unmarshal([]byte(numExtraData), &params)

									emailData["Parameters"] = params
								}

								if len(attachmentNames) > 0 {

									attachments := make([]map[string]interface{}, 0)
									for _, attachmentName := range attachmentNames {

										attachment := make(map[string]interface{})

										downloadUrl := fmt.Sprintf("http://%s/DVP/API/1.0.0.0/InternalFileService/File/DownloadLatest/%d/%d/%s", CreateHost(fileServiceHost, fileServicePort), tenant, company, attachmentName)
										attachment["name"] = attachmentName
										attachment["url"] = downloadUrl

										attachments = append(attachments, attachment)
									}

									emailData["attachments"] = attachments

								}

								publishData, pubDataConvErr := json.Marshal(emailData)
								fmt.Println("Email Pub data: ", string(publishData))
								if pubDataConvErr == nil {
									fmt.Println("Start Publish to rabbitMQ")
									RabbitMQPublish("EMAILOUT", publishData)

									SetSessionInfo(campaignId, sessionId, "Reason", "dial_success")
									SetSessionInfo(campaignId, sessionId, "DialerStatus", "dial_success")
								} else {
									SetSessionInfo(campaignId, sessionId, "Reason", "not_specified")
									SetSessionInfo(campaignId, sessionId, "DialerStatus", "dial_failed")
								}

							} else {
								fmt.Println("No Tamplate Found")
								SetSessionInfo(campaignId, sessionId, "Reason", "No Tamplate Found")
								SetSessionInfo(campaignId, sessionId, "DialerStatus", "dial_failed")
							}

							go UploadSessionInfo(campaignId, sessionId)
						}

						dialRateStr := strconv.Itoa(60000 / maxCampaignChannelLimit)
						dialRate, _ := time.ParseDuration(dialRateStr + "ms")
						time.Sleep(dialRate)
						break

					case "SMS":

						color.Yellow(fmt.Sprintf("=============== SMS DIALING - Campaign : %s ====================", campaignId))

						templates := RequestCampaignAddtionalData(company, tenant, campaignId, "BLAST", "SMS", "TEMPLATE")

						number, _, numExtraData, _, _, _, _ := GetNumberToDial(company, tenant, campaignId, camScheduleId, "")
						if number == "" {
							numberCount := GetNumberCount(company, tenant, campaignId, camScheduleId)
							if numberCount == 0 {
								SetCampaignStatus(campaignId, "PauseByDialer", company, tenant)
								return
							}
						} else {
							color.Yellow(fmt.Sprintf("=============== SMS DIALING - Number Found : %s ====================", number))
							uuidV4 := uuid.NewV4()
							sessionId := uuidV4.String()
							smsData := make(map[string]interface{})

							smsData["company"] = company
							smsData["tenant"] = tenant
							smsData["to"] = number
							smsData["from"] = "facetone"
							smsData["subject"] = campaignName

							InitiateSessionInfo(company, tenant, 240, "Campaign", "SMS", "BlastDial", "1", campaignId, scheduleId, campaignName, sessionId, number, "start", "dial_start", time.Now().UTC().Format(layout4), resourceServerInfos.ResourceServerId, nil, nil, "", "", "")

							if len(templates) > 0 {
								smsData["template"] = templates[0]

								if numExtraData != "" {
									var params map[string]interface{}
									json.Unmarshal([]byte(numExtraData), &params)

									smsData["Parameters"] = params
								}

								publishData, pubDataConvErr := json.Marshal(smsData)
								fmt.Println("SMS Pub data: ", string(publishData))
								if pubDataConvErr == nil {
									fmt.Println("Start Publish to rabbitMQ")
									whitemagenta := color.New(color.FgWhite).Add(color.BgMagenta)
									whitemagenta.Println(fmt.Sprintf("=============== SMS DIALING - Publishing : %s ====================", campaignId))
									RabbitMQPublish("SMSOUT", publishData)
									SetSessionInfo(campaignId, sessionId, "Reason", "dial_success")
									SetSessionInfo(campaignId, sessionId, "DialerStatus", "dial_success")
								} else {
									SetSessionInfo(campaignId, sessionId, "Reason", "not_specified")
									SetSessionInfo(campaignId, sessionId, "DialerStatus", "dial_failed")
								}

							} else {
								color.Yellow(fmt.Sprintf("=============== SMS DIALING - Template Not Found : %s ====================", campaignId))
								SetSessionInfo(campaignId, sessionId, "Reason", "No Template Found")
								SetSessionInfo(campaignId, sessionId, "DialerStatus", "dial_failed")
							}

							go UploadSessionInfo(campaignId, sessionId)
						}

						dialRateStr := strconv.Itoa(60000 / maxCampaignChannelLimit)
						dialRate, _ := time.ParseDuration(dialRateStr + "ms")
						time.Sleep(dialRate)
						break

					default:
						break

					}

				} else {
					color.Yellow("======== APPONITMENT ENDED - STATE CHANGED TO PAUSEDBYDIALER ========")
					SetCampaignStatus(campaignId, "PauseByDialer", company, tenant)
					SetCampChannelMaxLimitDirect(campaignId, "0")
					return
				}
			} else {
				switch campStatus {
				case "Stop":
					SetCampaignStatus(campaignId, "Stop", company, tenant)
					RemoveCampaignFromDialer(campaignId, company, tenant)
					return
				case "Pause":
					SetCampaignStatus(campaignId, "Pause", company, tenant)
					return
				case "End":
					SetCampaignStatus(campaignId, "End", company, tenant)
					RemoveCampaignFromDialer(campaignId, company, tenant)
					return
				case "PauseByDialer":
					SetCampaignStatus(campaignId, "PauseByDialer", company, tenant)
					return
				default:
					fmt.Println("Start Set ForceFullyStop: Current State:: ", campStatus)
					SetCampaignStatus(campaignId, "ForceFullyStop", company, tenant)
					return
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	} else {
		color.Yellow(fmt.Sprintf("APPOINTMENT OR RESOURCE SERVER NOT FOUND - SETTING CAMPAIGN : %s STATUS TO WAITING FOR APPOINTMENT", campaignId))
		SetCampaignStatus(campaignId, "Waiting for Appoinment", company, tenant)
		return
	}
}
