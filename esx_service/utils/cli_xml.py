# Copyright 2016 VMware, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

# A small module for creating vmdkops admin cli command outputs in xml format

import xml.dom.minidom

# top level tag in output requires esxcli namespace declared
ESXCLI_NS = "http://www.vmware.com/Products/ESX/6.0/esxcli/"

LIST_TAG = "list"
STRUCT_TAG = "structure"
STRUCT_NAME = "vdvs"
STR_TAG = "string"
FIELD_TAG = "field"
OUTPUT_TAG = "output"


def createDocXML():
    """
    Generates the main output object with namespace attribute defined
    """
    doc = xml.dom.minidom.Document()
    output_elem = doc.createElementNS(ESXCLI_NS, OUTPUT_TAG)
    output_elem.setAttribute("xmlns", ESXCLI_NS)
    doc.appendChild(output_elem)
    return doc, output_elem


def createFieldList(doc, struct_elem, header, text):
    """
    Generates a field tag against header value and appends it to parent
    struct element
    """
    field_elem = doc.createElement(FIELD_TAG)
    field_elem.setAttribute("name", header)
    struct_elem.appendChild(field_elem)
    str_elem = doc.createElement(STR_TAG)
    field_elem.appendChild(str_elem)
    str_elem.appendChild(doc.createTextNode(text))


def createStruct(doc, list_elem):
    """
    Generates a struct tag element and appends it to the list element
    Returns the struct tag element
    """
    struct_elem = doc.createElement(STRUCT_TAG)
    struct_elem.setAttribute("typeName", STRUCT_NAME)
    list_elem.appendChild(struct_elem)
    return struct_elem


def create(header, rows):
    """
    Generates xml with output tag. It internally contains a list of tags
    representating each row in rows against the header
    Returns formatted xml doc object
    """
    doc, output_elem = createDocXML()
    list_elem = doc.createElement(LIST_TAG)
    list_elem.setAttribute("type", STRUCT_TAG)
    output_elem.appendChild(list_elem)

    if not rows:
        struct_elem = createStruct(doc, list_elem)
        for h_iter in header:
            createFieldList(doc, struct_elem, h_iter, "")

    else:
        for r_iter in rows:
            struct_elem = createStruct(doc, list_elem)
            for i, cell in enumerate(r_iter):
                createFieldList(doc, struct_elem, header[i], cell)

    return doc.toprettyxml()

def createMessage(message):
    """
    Generates xml with output tag with a string field for representing message
    Returns formatted xml doc object
    """
    doc, output_elem = createDocXML()
    str_elem = doc.createElement(STR_TAG)
    output_elem.appendChild(str_elem)
    str_elem.appendChild(doc.createTextNode(message))
    return doc.toprettyxml()