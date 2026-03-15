'use client'

import { BlockquotePlugin, H1Plugin, H2Plugin, H3Plugin } from "@platejs/basic-nodes/react"
import { CodeBlockPlugin, CodeLinePlugin } from "@platejs/code-block/react"
import { ParagraphPlugin } from "platejs/react"

import {
  CollabBlockquoteElement,
  CollabCodeBlockElement,
  CollabCodeLineElement,
  CollabH1Element,
  CollabH2Element,
  CollabH3Element,
  CollabParagraphElement,
} from "@/components/editor/collab-editor-elements"

export const CollabBlocksKit = [
  ParagraphPlugin.withComponent(CollabParagraphElement),
  H1Plugin.withComponent(CollabH1Element),
  H2Plugin.withComponent(CollabH2Element),
  H3Plugin.withComponent(CollabH3Element),
  BlockquotePlugin.withComponent(CollabBlockquoteElement),
  CodeBlockPlugin.withComponent(CollabCodeBlockElement),
  CodeLinePlugin.withComponent(CollabCodeLineElement),
]
