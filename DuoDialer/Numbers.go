package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	//"bufio"
	//"os"
)

func GetNumbersFromNumberBase(company, tenant, numberLimit int, campaignId, camScheduleId string) []string {
	numbers := make([]string, 0)
	pageKey := fmt.Sprintf("PhoneNumberPage:%d:%d:%s:%s", company, tenant, campaignId, camScheduleId)
	pageNumberToRequest := RedisIncr(pageKey)
	fmt.Println("pageNumber: ", pageNumberToRequest)
	/*if pageNumberToRequest == 1 {
		numbers = append(numbers, "0773795991")
		numbers = append(numbers, "0773795992")
		numbers = append(numbers, "0773795993")
		numbers = append(numbers, "0773795994")
		numbers = append(numbers, "0773795995")
		numbers = append(numbers, "0773795996")
		numbers = append(numbers, "0773795997")
		numbers = append(numbers, "0773795998")
		numbers = append(numbers, "0773795999")
		numbers = append(numbers, "0773795990")
	}*/

	/*if pageNumberToRequest == 1 {
		file, err := os.Open("D:\\Duo Projects\\Version 5.1\\Documents\\GolangProjects\\CampaignManager\\NumberList4.txt")
		if err != nil {
			//log.Fatal(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			RedisListLpush("CampaignNumbers:4:2:1:1", scanner.Text())
			fmt.Println(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			//log.Fatal(err)
		}
	}*/

	// Get phone number from campign service and append
	authToken := fmt.Sprintf("%d#%d", company, tenant)
	fmt.Println("Start GetPhoneNumbers Auth: ", authToken, " CampaignId: ", campaignId, " camScheduleId: ", camScheduleId)
	client := &http.Client{}

	request := fmt.Sprintf("%s/CampaignManager/NumberUpload/%s/%s/%d/%d", campaignService, campaignId, camScheduleId, numberLimit, pageNumberToRequest)
	fmt.Println("Start GetPhoneNumbers request: ", request)
	req, _ := http.NewRequest("GET", request, nil)
	req.Header.Add("Authorization", authToken)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return numbers
	}
	defer resp.Body.Close()

	response, _ := ioutil.ReadAll(resp.Body)

	var phoneNumberResult PhoneNumberResult
	json.Unmarshal(response, &phoneNumberResult)
	if phoneNumberResult.IsSuccess == true {
		for _, numRes := range phoneNumberResult.Result {
			numbers = append(numbers, numRes.CampContactInfo.ContactId)
		}
	}
	return numbers
}

func LoadNumbers(company, tenant, numberLimit int, campaignId, camScheduleId string) {
	listId := fmt.Sprintf("CampaignNumbers:%d:%d:%s:%s", company, tenant, campaignId, camScheduleId)
	numbers := GetNumbersFromNumberBase(company, tenant, numberLimit, campaignId, camScheduleId)
	if len(numbers) == 0 {
		numLoadingStatusKey := fmt.Sprintf("PhoneNumberLoading:%d:%d:%s:%s", company, tenant, campaignId, camScheduleId)
		RedisSet(numLoadingStatusKey, "done")
	} else {
		numLoadingStatusKey := fmt.Sprintf("PhoneNumberLoading:%d:%d:%s:%s", company, tenant, campaignId, camScheduleId)
		RedisSet(numLoadingStatusKey, "waiting")
		for _, number := range numbers {
			RedisListRpush(listId, number)
		}
	}
}

func LoadInitialNumberSet(company, tenant int, campaignId, camScheduleId string) {
	numLoadingStatusKey := fmt.Sprintf("PhoneNumberLoading:%d:%d:%s:%s", company, tenant, campaignId, camScheduleId)
	LoadNumbers(company, tenant, 1000, campaignId, camScheduleId)
	RedisSet(numLoadingStatusKey, "waiting")
}

func GetNumberToDial(company, tenant int, campaignId, camScheduleId string) string {
	listId := fmt.Sprintf("CampaignNumbers:%d:%d:%s:%s", company, tenant, campaignId, camScheduleId)
	numLoadingStatusKey := fmt.Sprintf("PhoneNumberLoading:%d:%d:%s:%s", company, tenant, campaignId, camScheduleId)
	numberCount := RedisListLlen(listId)
	numLoadingStatus := RedisGet(numLoadingStatusKey)

	if numLoadingStatus == "waiting" {
		if numberCount < 500 {
			LoadNumbers(company, tenant, 500, campaignId, camScheduleId)
		}
	} else if numLoadingStatus == "done" && numberCount == 0 {
		pageKey := fmt.Sprintf("PhoneNumberPage:%d:%d:%s:%s", company, tenant, campaignId, camScheduleId)
		RedisRemove(numLoadingStatusKey)
		RedisRemove(pageKey)
	}
	return RedisListLpop(listId)
}

func GetNumberCount(company, tenant int, campaignId, camScheduleId string) int {
	listId := fmt.Sprintf("CampaignNumbers:%d:%d:%s:%s", company, tenant, campaignId, camScheduleId)
	return RedisListLlen(listId)
}

func RemoveNumbers(company, tenant int, campaignId string) {
	searchKey := fmt.Sprintf("CampaignNumbers:%d:%d:%s:*", company, tenant, campaignId)
	relatedNumberList := RedisSearchKeys(searchKey)
	for _, key := range relatedNumberList {
		RedisRemove(key)
	}
}