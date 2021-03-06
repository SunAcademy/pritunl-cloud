package uhandlers

import (
	"github.com/gin-gonic/gin"
	"github.com/pritunl/pritunl-cloud/database"
	"github.com/pritunl/pritunl-cloud/datacenter"
	"github.com/pritunl/pritunl-cloud/utils"
	"github.com/pritunl/pritunl-cloud/zone"
	"gopkg.in/mgo.v2/bson"
)

func zonesGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	userOrg := c.MustGet("organization").(bson.ObjectId)

	dcIds, err := datacenter.DistinctOrg(db, userOrg)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	zones, err := zone.GetAllDatacenters(db, dcIds)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	c.JSON(200, zones)
}
