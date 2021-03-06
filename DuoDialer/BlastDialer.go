package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/fatih/color"
)

func DialNumber(company, tenant int, resourceServer ResourceServerInfo, campaignId, scheduleId, campaignName, uuid, fromNumber, trunkCode, phoneNumber, xGateway, tryCount, extention string, integrationData *IntegrationConfig, contacts *[]Contact, thirdpartyreference, businessUnit string) {
	DialerLog(fmt.Sprintf("Start DialNumber: %s:%s:%s:%s:%s:%s", uuid, fromNumber, trunkCode, phoneNumber, extention, xGateway))
	customCompanyStr := fmt.Sprintf("%d_%d", company, tenant)

	var param string
	if xGateway != "" {
		param = fmt.Sprintf(" {DVP_CUSTOM_PUBID=%s,CampaignId=%s,CampaignName='%s',tenantid=%d,companyid=%d,CustomCompanyStr=%s,OperationType=Dialer,DVP_ACTION_CAT=DIALER,DVP_OPERATION_CAT=CUSTOMER,DVP_ADVANCED_OP_ACTION=BLAST,DVP_CALL_DIRECTION=outbound,CALL_LEG_TYPE=CUSTOMER,return_ring_ready=true,ignore_early_media=false,origination_uuid=%s,origination_caller_id_number=%s,originate_timeout=30,sip_h_X-Gateway=%s,dialer_from_number=%s,dialer_to_number=%s}", subChannelName, campaignId, campaignName, tenant, company, customCompanyStr, uuid, fromNumber, xGateway, fromNumber, phoneNumber)
	} else {
		param = fmt.Sprintf(" {DVP_CUSTOM_PUBID=%s,CampaignId=%s,CampaignName='%s',tenantid=%d,companyid=%d,CustomCompanyStr=%s,OperationType=Dialer,DVP_ACTION_CAT=DIALER,DVP_OPERATION_CAT=CUSTOMER,DVP_ADVANCED_OP_ACTION=BLAST,DVP_CALL_DIRECTION=outbound,CALL_LEG_TYPE=CUSTOMER,return_ring_ready=true,ignore_early_media=false,origination_uuid=%s,origination_caller_id_number=%s,originate_timeout=30,dialer_from_number=%s,dialer_to_number=%s}", subChannelName, campaignId, campaignName, tenant, company, customCompanyStr, uuid, fromNumber, fromNumber, phoneNumber)
	}
	furl := fmt.Sprintf("sofia/gateway/%s/%s %s", trunkCode, phoneNumber, extention)
	data := fmt.Sprintf(" xml %d_%d_dialer", tenant, company)

	strTenant := strconv.Itoa(tenant)
	strCompany := strconv.Itoa(company)

	IncrConcurrentChannelCount(resourceServer.ResourceServerId, campaignId)
	AddCampaignCallsRealtime(phoneNumber, tryCount, "DIALING", strTenant, strCompany, campaignId, uuid)
	IncrCampaignDialCount(company, tenant, campaignId)
	InitiateSessionInfo(company, tenant, 240, "Campaign", "Dialer", "BlastDial", tryCount, campaignId, scheduleId, campaignName, uuid, phoneNumber, "start", "dial_start", time.Now().UTC().Format(layout4), resourceServer.ResourceServerId, integrationData, contacts, "", thirdpartyreference, businessUnit)

	redwhite := color.New(color.FgRed).Add(color.BgWhite)
	redwhite.Println(fmt.Sprintf("DIALING OUT CALL - BLAST CAMPAIGN : %s | NUMBER : %s", campaignName, phoneNumber))
	//PublishCampaignCallCounts(uuid, "DIALED", strCompany, strTenant, campaignId)
	SetSessionInfo(campaignId, uuid, "IsDialed", "TRUE")
	dashboardparam2 := "BASIC"
	tryCountInt, _ := strconv.Atoi(tryCount)
	if tryCountInt > 1 {
		dashboardparam2 = "CALLBACK"
	}
	PublishCampaignCallCounts(uuid, "DIALING", strCompany, strTenant, campaignId, dashboardparam2)
	resp, err := Dial(resourceServer.Url, param, furl, data)
	HandleDialResponse(resp, err, resourceServer, campaignId, uuid)
}
