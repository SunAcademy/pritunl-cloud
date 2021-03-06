package uhandlers

import (
	"fmt"
	"github.com/dropbox/godropbox/container/set"
	"github.com/gin-gonic/gin"
	"github.com/pritunl/pritunl-cloud/database"
	"github.com/pritunl/pritunl-cloud/datacenter"
	"github.com/pritunl/pritunl-cloud/demo"
	"github.com/pritunl/pritunl-cloud/event"
	"github.com/pritunl/pritunl-cloud/utils"
	"github.com/pritunl/pritunl-cloud/vpc"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"strings"
)

type vpcData struct {
	Id         bson.ObjectId `json:"id"`
	Name       string        `json:"name"`
	Network    string        `json:"network"`
	Datacenter bson.ObjectId `json:"datacenter"`
	Routes     []*vpc.Route  `json:"routes"`
	LinkUris   []string      `json:"link_uris"`
}

type vpcsData struct {
	Vpcs  []*vpc.Vpc `json:"vpcs"`
	Count int        `json:"count"`
}

func vpcPut(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	userOrg := c.MustGet("organization").(bson.ObjectId)
	data := &vpcData{}

	vpcId, ok := utils.ParseObjectId(c.Param("vpc_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	exists, err := datacenter.ExistsOrg(db, userOrg, data.Datacenter)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}
	if !exists {
		utils.AbortWithStatus(c, 405)
		return
	}

	vc, err := vpc.GetOrg(db, userOrg, vpcId)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if vc.Organization != userOrg {
		utils.AbortWithStatus(c, 405)
		return
	}

	vc.Name = data.Name
	vc.Routes = data.Routes
	vc.LinkUris = data.LinkUris

	fields := set.NewSet(
		"state",
		"name",
		"routes",
		"link_uris",
	)

	errData, err := vc.Validate(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	err = vc.CommitFields(db, fields)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.PublishDispatch(db, "vpc.change")

	vc.Json()

	c.JSON(200, vc)
}

func vpcPost(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	userOrg := c.MustGet("organization").(bson.ObjectId)
	data := &vpcData{
		Name: "New VPC",
	}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	exists, err := datacenter.ExistsOrg(db, userOrg, data.Datacenter)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}
	if !exists {
		utils.AbortWithStatus(c, 405)
		return
	}

	vc := &vpc.Vpc{
		Name:         data.Name,
		Network:      data.Network,
		Organization: userOrg,
		Datacenter:   data.Datacenter,
		Routes:       data.Routes,
		LinkUris:     data.LinkUris,
	}

	vc.GenerateVpcId()

	errData, err := vc.Validate(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	err = vc.Insert(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.PublishDispatch(db, "vpc.change")

	vc.Json()

	c.JSON(200, vc)
}

func vpcDelete(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	userOrg := c.MustGet("organization").(bson.ObjectId)

	vpcId, ok := utils.ParseObjectId(c.Param("vpc_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	exists, err := vpc.ExistsOrg(db, userOrg, vpcId)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}
	if !exists {
		utils.AbortWithStatus(c, 405)
		return
	}

	err = vpc.Remove(db, vpcId)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.PublishDispatch(db, "vpc.change")

	c.JSON(200, nil)
}

func vpcsDelete(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	userOrg := c.MustGet("organization").(bson.ObjectId)
	data := []bson.ObjectId{}

	err := c.Bind(&data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	for _, vpcId := range data {
		exists, e := vpc.ExistsOrg(db, userOrg, vpcId)
		if e != nil {
			utils.AbortWithError(c, 500, e)
			return
		}
		if !exists {
			utils.AbortWithStatus(c, 405)
			return
		}
	}

	err = vpc.RemoveMulti(db, data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.PublishDispatch(db, "vpc.change")

	c.JSON(200, nil)
}

func vpcGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	userOrg := c.MustGet("organization").(bson.ObjectId)

	vpcId, ok := utils.ParseObjectId(c.Param("vpc_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	vc, err := vpc.GetOrg(db, userOrg, vpcId)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	vc.Json()

	c.JSON(200, vc)
}

func vpcsGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	userOrg := c.MustGet("organization").(bson.ObjectId)

	if c.Query("names") == "true" {
		query := &bson.M{
			"organization": userOrg,
		}

		vpcs, err := vpc.GetAllNames(db, query)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		c.JSON(200, vpcs)
	} else {
		page, _ := strconv.Atoi(c.Query("page"))
		pageCount, _ := strconv.Atoi(c.Query("page_count"))

		query := bson.M{
			"organization": userOrg,
		}

		vpcId, ok := utils.ParseObjectId(c.Query("id"))
		if ok {
			query["_id"] = vpcId
		}

		name := strings.TrimSpace(c.Query("name"))
		if name != "" {
			query["name"] = &bson.M{
				"$regex":   fmt.Sprintf(".*%s.*", name),
				"$options": "i",
			}
		}

		network := strings.TrimSpace(c.Query("network"))
		if network != "" {
			query["network"] = network
		}

		dc, ok := utils.ParseObjectId(c.Query("datacenter"))
		if ok {
			query["datacenter"] = dc
		}

		vpcs, count, err := vpc.GetAllPaged(db, &query, page, pageCount)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		for _, vc := range vpcs {
			vc.Json()
		}

		data := &vpcsData{
			Vpcs:  vpcs,
			Count: count,
		}

		c.JSON(200, data)
	}
}
