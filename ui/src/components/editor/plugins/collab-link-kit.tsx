'use client'

import { LinkPlugin } from "@platejs/link/react"

import { CollabLinkElement } from "../collab-editor-elements"

export const CollabLinkKit = [LinkPlugin.withComponent(CollabLinkElement)]
