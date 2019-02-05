// BlastDialer project main.go
package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/DuoSoftware/gorest"
	"github.com/fatih/color"
	"github.com/rs/cors"
)

func errHndlr(err error) {
	if err != nil {
		fmt.Println("error:", err)
	}
}

func errHndlrNew(errorFrom, command string, err error) {
	if err != nil {
		fmt.Println("error:", errorFrom, ":: ", command, ":: ", err)
	}
}

func CheckTimeouts() {
	for {
		cblist := RedisHashGetAll("CALLBACK_TIMEOUTS")

		for cbKey, cbVal := range cblist {
			tNow := time.Now().Unix()
			tThen, _ := strconv.ParseInt(cbVal, 10, 64)

			prevTimeInt, _ := strconv.ParseInt(previewTimeout, 10, 64)

			timeDiff := tNow - tThen
			color.Cyan("TIME NOW : %d", tNow)
			color.Cyan("TIME THEN : %s", cbVal)
			color.Cyan("TIME DIFF : %d", timeDiff)

			if timeDiff > prevTimeInt {
				//Decrement Campaign Count
				hKey := fmt.Sprintf("sessionInfo:%s", cbKey)
				sessionInfo := RedisHashGetAll(hKey)
				DecrConcurrentChannelCount(sessionInfo["ResourceServerId"], sessionInfo["CampaignId"])
				RedisHashDelField("CALLBACK_TIMEOUTS", sessionInfo["CampaignId"]+":"+sessionInfo["SessionId"])
				SetSessionInfo(sessionInfo["CampaignId"], sessionInfo["SessionId"], "Reason", "callback_timeout")
				SetSessionInfo(sessionInfo["CampaignId"], sessionInfo["SessionId"], "DialerStatus", "failed")
				go UploadSessionInfo(sessionInfo["CampaignId"], sessionInfo["SessionId"])
			}

		}
		time.Sleep(10 * time.Second)
	}
}

func main() {

	//Innitiate configuration
	InitiateDuoDialer()

	//Initiate http api server for api calls
	go InitiateService()
	//Register Dialer on ARDS via service call
	go AddRequestServer()

	go EnableConsoleInput()

	go CheckTimeouts()

	//AddPhoneNumberToCallback("1", "1", "1", "1", "0112546969", "USER_BUSY")
	//MAIN THREAD
	for {
		//Get current campaign count
		onGoingCampaignCount := GetOnGoingCampaignCount()
		color.Blue(fmt.Sprintf("Ongoing Campaign Count : %d - Campaign Limit : %d", onGoingCampaignCount, campaignLimit))
		if onGoingCampaignCount < campaignLimit {
			//Request for more campaigns
			campaigns := RequestCampaign(campaignLimit - onGoingCampaignCount)
			for _, campaign := range campaigns {
				//Adding Campaign to Dialer
				AddCampaignToDialer(campaign)
			}
		}

		if onGoingCampaignCount > 0 {

			runningCampaigns := GetAllRunningCampaign()
			for _, campaign := range runningCampaigns {

				StartLocation, _ := time.LoadLocation(campaign.CampConfigurations.StartTimeZone)
				EndLocation, _ := time.LoadLocation(campaign.CampConfigurations.EndTimeZone)
				tmStartLocation := time.Now().In(StartLocation)
				tmEndLocation := time.Now().In(EndLocation)

				campIdStr := strconv.Itoa(campaign.CampaignId)

				campaignStartDate := campaign.CampConfigurations.StartDate
				campaignEndDate := campaign.CampConfigurations.EndDate

				go ClearTimeoutChannels(campIdStr)

				if campaignEndDate.Before(tmEndLocation) {
					DialerLog(fmt.Sprintf("campaignEndDate before: %s", tmEndLocation.String()))
					color.Yellow("ENDING CAMPAIGN DUE TO TIME EXPIRING - SET STATUS TO END")
					SetCampaignStatus(campIdStr, "End", campaign.CompanyId, campaign.TenantId)
					RemoveCampaignFromDialer(campIdStr, campaign.CompanyId, campaign.TenantId)
				} else {

					//campStatus := GetCampaignStatus(campIdStr, campaign.CompanyId, campaign.TenantId)

					campStatus := UpdateCampaignStatus(campaign.CompanyId, campaign.TenantId, campIdStr)
					color.Red(fmt.Sprintf("%s : %s", campaign.CampaignName, campStatus))
					DialerLog(fmt.Sprintf("campStatus: %s", campStatus))

					if campStatus == "Resume" || campStatus == "Start" || campStatus == "PauseByDialer" || campStatus == "Waiting for Appoinment" {
						//tempCampaignStartDate, _ := time.Parse(layout2, campaign.CampConfigurations.StartDate)
						//tempCampaignEndDate, _ := time.Parse(layout2, campaign.CampConfigurations.EndDate)

						if campStatus == "Resume" {
							UpdateCampaignStartStatus(campaign.CompanyId, campaign.TenantId, campIdStr)
						}

						//campaignStartDate := time.Date(tempCampaignStartDate.Year(), tempCampaignStartDate.Month(), tempCampaignStartDate.Day(), 0, 0, 0, 0, location)
						//campaignEndDate := time.Date(tempCampaignEndDate.Year(), tempCampaignEndDate.Month(), tempCampaignEndDate.Day(), 0, 0, 0, 0, location)

						color.Cyan(fmt.Sprintf("Integration Data : %s", campaign.CampConfigurations.IntegrationData))

						DialerLog(fmt.Sprintf("Check Campaign: %s", campIdStr))
						DialerLog(fmt.Sprintf("campaignStartDate: %s", campaign.CampConfigurations.StartDate.String()))
						DialerLog(fmt.Sprintf("campaignEndDate: %s", campaign.CampConfigurations.EndDate.String()))

						if campaignStartDate.Before(tmStartLocation) && campaignEndDate.After(tmEndLocation) {
							DialerLog(fmt.Sprintf("Continue campaign: %s", campIdStr))
							if len(campaign.CampScheduleInfo) > 0 {

								for _, schedule := range campaign.CampScheduleInfo {
									scheduleId := strconv.Itoa(schedule.ScheduleId)
									camScheduleId := strconv.Itoa(schedule.CamScheduleId)
									//Start Dialing the campaign
									color.Green(fmt.Sprintf("====== CAMPAIGN %s READY TO START ======", campaign.CampaignName))
									color.Cyan(fmt.Sprintf("CAMPAIGN : %v", campaign))
									tempCamp := campaign
									go StartCampaign(campIdStr, campaign.CampaignName, campaign.DialoutMechanism, campaign.CampaignChannel, campaign.Class, campaign.Type, campaign.Category, scheduleId, camScheduleId, "*", campaign.Extensions, campaign.CampConfigurations.Caller, campaign.CompanyId, campaign.TenantId, campaign.CampConfigurations.ChannelConcurrency, &tempCamp.CampConfigurations.IntegrationData, campaign.CampConfigurations.NumberLoadingMethod)
								}
							}
						}
					} else {
						switch campStatus {
						case "Stop":
							SetCampaignStatus(campIdStr, "Stop", campaign.CompanyId, campaign.TenantId)
							RemoveCampaignFromDialer(campIdStr, campaign.CompanyId, campaign.TenantId)
							break
						case "End":
							SetCampaignStatus(campIdStr, "End", campaign.CompanyId, campaign.TenantId)
							RemoveCampaignFromDialer(campIdStr, campaign.CompanyId, campaign.TenantId)
							break
						default:
							break
						}
					}
				}
			}
		}

		time.Sleep(campaignRequestFrequency * time.Second)
	}
}

func InitiateService() {
	//jwtMiddleware := loadJwtMiddleware()
	gorest.RegisterService(new(DVP))
	//http.Handle("/", gorest.Handle())
	//app := jwtMiddleware.Handler(gorest.Handle())
	//addr := fmt.Sprintf(":%s", port)
	//fmt.Println(addr)
	//http.ListenAndServe(addr, app)

	c := cors.New(cors.Options{
		AllowedHeaders: []string{"accept", "authorization"},
	})
	handler := c.Handler(gorest.Handle())
	addr := fmt.Sprintf(":%s", port)
	s := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	//s.SetKeepAlivesEnabled(false)
	s.ListenAndServe()
}
