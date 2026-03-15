'use client'

import { IndentPlugin } from "@platejs/indent/react"
import { ListPlugin } from "@platejs/list/react"

import { CollabListElement } from "../collab-editor-elements"

export const CollabListsKit = [
  IndentPlugin,
  ListPlugin.configure({
    render: {
      belowNodes: (props) => {
        if (!props.element.listStyleType) {
          return
        }

        return (listProps) => <CollabListElement {...listProps} />
      },
    },
  }),
]
