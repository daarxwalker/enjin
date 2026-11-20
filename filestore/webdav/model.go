package webdav

import "encoding/xml"

type multistatusDAVRow struct {
	Responses []responseDAVRow `xml:"DAV: response"`
}

type responseDAVRow struct {
	Href      string           `xml:"DAV: href"`
	Propstats []propstatDAVRow `xml:"DAV: propstat"`
}

type propstatDAVRow struct {
	Status string     `xml:"DAV: status"`
	Prop   propDAVRow `xml:"DAV: prop"`
}

type propDAVRow struct {
	ContentLength string             `xml:"DAV: getcontentlength"`
	ContentType   string             `xml:"DAV: getcontenttype"`
	ETag          string             `xml:"DAV: getetag"`
	LastModified  string             `xml:"DAV: getlastmodified"`
	ResourceType  resourceTypeDAVRow `xml:"DAV: resourcetype"`
}

// resourceTypeDAVRow keeps the name of the collection element: encoding/xml
// fills an xml.Name field with the element it finds, so an empty Local means
// the resource is no collection.
type resourceTypeDAVRow struct {
	Collection xml.Name `xml:"DAV: collection"`
}
