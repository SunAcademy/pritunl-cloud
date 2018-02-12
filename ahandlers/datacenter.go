package ahandlers

import (
	"github.com/dropbox/godropbox/container/set"
	"github.com/gin-gonic/gin"
	"github.com/pritunl/pritunl-cloud/database"
	"github.com/pritunl/pritunl-cloud/datacenter"
	"github.com/pritunl/pritunl-cloud/demo"
	"github.com/pritunl/pritunl-cloud/event"
	"github.com/pritunl/pritunl-cloud/utils"
	"gopkg.in/mgo.v2/bson"
)

type datacenterData struct {
	Id            bson.ObjectId   `json:"id"`
	Name          string          `json:"name"`
	Organizations []bson.ObjectId `json:"organizations"`
}

func datacenterPut(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &datacenterData{}

	dcId, ok := utils.ParseObjectId(c.Param("dc_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	dc, err := datacenter.Get(db, dcId)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	dc.Name = data.Name
	dc.Organizations = data.Organizations

	fields := set.NewSet(
		"name",
		"organizations",
	)

	errData, err := dc.Validate(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	err = dc.CommitFields(db, fields)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.PublishDispatch(db, "datacenter.change")

	c.JSON(200, dc)
}

func datacenterPost(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &datacenterData{
		Name: "New Datacenter",
	}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	dc := &datacenter.Datacenter{
		Name:          data.Name,
		Organizations: data.Organizations,
	}

	errData, err := dc.Validate(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	err = dc.Insert(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.PublishDispatch(db, "datacenter.change")

	c.JSON(200, dc)
}

func datacenterDelete(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)

	dcId, ok := utils.ParseObjectId(c.Param("dc_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	err := datacenter.Remove(db, dcId)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.PublishDispatch(db, "datacenter.change")

	c.JSON(200, nil)
}

func datacenterGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)

	dcId, ok := utils.ParseObjectId(c.Param("dc_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	dc, err := datacenter.Get(db, dcId)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	c.JSON(200, dc)
}

func datacentersGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)

	dcs, err := datacenter.GetAll(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	c.JSON(200, dcs)
}