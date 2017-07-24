// BlastDialer project main.go
package main

import (
	"encoding/json"
	"fmt"
	"github.com/DuoSoftware/gorest"
	"github.com/rs/cors"
	"net/http"
	"strconv"
	"time"
)

func errHndlr(err error) {
	if err != nil {
		fmt.Println("error:", err)
	}
}

func main() {
	params := make(map[string]interface{})
	attachment := make(map[string]interface{})
	attachments := make([]map[string]interface{}, 0)

	params["name"] = "Heshan Indika"
	params["Code"] = "123456"

	attachment["name"] = "sample.csv"
	attachment["url"] = "http://abc.app.veery.cloud/rrr/eee"

	attachments = append(attachments, attachment)

	data := make(map[string]interface{})
	data["to"] = "heshan.i@duosoftware.com"
	data["subject"] = "Test Mail"
	data["Parameters "] = params
	data["attachments  "] = attachments

	aaa, _ := json.Marshal(data)
	fmt.Println(string(aaa))

	InitiateDuoDialer()

	go InitiateService()
	go AddRequestServer()

	//AddPhoneNumberToCallback("1", "1", "1", "1", "0112546969", "USER_BUSY")
	for {
		onGoingCampaignCount := GetOnGoingCampaignCount()
		if onGoingCampaignCount < campaignLimit {
			campaigns := RequestCampaign(campaignLimit - onGoingCampaignCount)
			for _, campaign := range campaigns {
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
					fmt.Println("campaignEndDate before: ", tmEndLocation.String())
					SetCampaignStatus(campIdStr, "End", campaign.CompanyId, campaign.TenantId)
					RemoveCampaignFromDialer(campIdStr, campaign.CompanyId, campaign.TenantId)
				} else {

					campStatus := GetCampaignStatus(campIdStr, campaign.CompanyId, campaign.TenantId)
					fmt.Println("campStatus: ", campStatus)
					UpdateCampaignStatus(campaign.CompanyId, campaign.TenantId, campIdStr)

					if campStatus == "Resume" || campStatus == "Start" || campStatus == "PauseByDialer" || campStatus == "Waiting for Appoinment" {
						//tempCampaignStartDate, _ := time.Parse(layout2, campaign.CampConfigurations.StartDate)
						//tempCampaignEndDate, _ := time.Parse(layout2, campaign.CampConfigurations.EndDate)

						if campStatus == "Resume" {
							UpdateCampaignStartStatus(campaign.CompanyId, campaign.TenantId, campIdStr)
						}

						//campaignStartDate := time.Date(tempCampaignStartDate.Year(), tempCampaignStartDate.Month(), tempCampaignStartDate.Day(), 0, 0, 0, 0, location)
						//campaignEndDate := time.Date(tempCampaignEndDate.Year(), tempCampaignEndDate.Month(), tempCampaignEndDate.Day(), 0, 0, 0, 0, location)

						fmt.Println("Check Campaign: ", campIdStr)
						fmt.Println("campaignStartDate: ", campaign.CampConfigurations.StartDate.String())
						fmt.Println("campaignEndDate: ", campaign.CampConfigurations.EndDate.String())

						if campaignStartDate.Before(tmStartLocation) && campaignEndDate.After(tmEndLocation) {
							fmt.Println("Continue campaign: ", campIdStr)
							if len(campaign.CampScheduleInfo) > 0 {

								for _, schedule := range campaign.CampScheduleInfo {
									scheduleId := strconv.Itoa(schedule.ScheduleId)
									camScheduleId := strconv.Itoa(schedule.CamScheduleId)
									go StartCampaign(campIdStr, campaign.CampaignName, campaign.DialoutMechanism, campaign.CampaignChannel, campaign.Class, campaign.Type, campaign.Category, scheduleId, camScheduleId, "*", campaign.Extensions, campaign.CampConfigurations.Caller, campaign.CompanyId, campaign.TenantId, campaign.CampConfigurations.ChannelConcurrency)
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
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	//s.SetKeepAlivesEnabled(false)
	s.ListenAndServe()
}
