'use client'

import type { PlateElementProps, PlateLeafProps } from "platejs/react"

import { isOrderedList } from "@platejs/list"
import { useTodoListElement, useTodoListElementState } from "@platejs/list/react"
import { Checkbox } from "@/components/ui/checkbox"
import { BlockquoteElement } from "@/components/ui/blockquote-node"
import { H1Element, H2Element, H3Element } from "@/components/ui/heading-node"
import { ParagraphElement } from "@/components/ui/paragraph-node"
import { cn } from "@/lib/utils"
import { PlateElement, PlateLeaf } from "platejs/react"

import { TODO_LIST_STYLE_TYPE } from "./collab-editor-constants"

type CollabElement = PlateElementProps["element"] & {
  checked?: boolean
  listStart?: number
  listStyleType?: string
  url?: string
}

function TodoParagraphElement(props: PlateElementProps) {
  const element = props.element as CollabElement
  const state = useTodoListElementState({ element })
  const { checkboxProps } = useTodoListElement(state)

  return (
    <PlateElement
      {...props}
      as="p"
      className={cn(
        "my-1 flex items-start gap-2",
        element.checked && "text-muted-foreground line-through"
      )}
    >
      <Checkbox
        checked={checkboxProps.checked}
        className="mt-1"
        onCheckedChange={(value) => checkboxProps.onCheckedChange(value === true)}
        onMouseDown={checkboxProps.onMouseDown}
      />
      <span className="min-w-0 flex-1">{props.children}</span>
    </PlateElement>
  )
}

export function CollabParagraphElement(props: PlateElementProps) {
  const element = props.element as CollabElement

  if (element.listStyleType === TODO_LIST_STYLE_TYPE) {
    return <TodoParagraphElement {...props} />
  }

  return <ParagraphElement {...props} />
}

export function CollabH1Element(props: PlateElementProps) {
  return <H1Element {...props} />
}

export function CollabH2Element(props: PlateElementProps) {
  return <H2Element {...props} />
}

export function CollabH3Element(props: PlateElementProps) {
  return <H3Element {...props} />
}

export function CollabBlockquoteElement(props: PlateElementProps) {
  return <BlockquoteElement {...props} />
}

export function CollabCodeBlockElement(props: PlateElementProps) {
  return (
    <PlateElement
      {...props}
      as="pre"
      className="my-2 overflow-x-auto rounded-md bg-muted px-4 py-3 font-mono text-sm"
    >
      <code>{props.children}</code>
    </PlateElement>
  )
}

export function CollabCodeLineElement(props: PlateElementProps) {
  return (
    <PlateElement
      {...props}
      as="div"
      className="min-h-5 whitespace-pre-wrap break-words"
    />
  )
}

export function CollabLinkElement(props: PlateElementProps) {
  const element = props.element as CollabElement

  return (
    <PlateElement {...props} as="span" className="font-medium text-sky-700">
      <a
        className="underline underline-offset-4"
        href={element.url}
        rel="noreferrer noopener"
        target="_blank"
      >
        {props.children}
      </a>
    </PlateElement>
  )
}

export function CollabColorLeaf(props: PlateLeafProps) {
  const style = {
    backgroundColor:
      typeof props.leaf.backgroundColor === "string"
        ? props.leaf.backgroundColor
        : undefined,
    color: typeof props.leaf.color === "string" ? props.leaf.color : undefined,
  }

  return <PlateLeaf {...props} style={style} />
}

export function CollabListElement(props: PlateElementProps) {
  const element = props.element as CollabElement
  const isTodoList = element.listStyleType === TODO_LIST_STYLE_TYPE
  const ListTag = isTodoList ? "ul" : isOrderedList(element) ? "ol" : "ul"
  const style = isTodoList
    ? undefined
    : {
        listStyleType: element.listStyleType,
      }

  return (
    <ListTag
      className={cn(
        "my-1 ml-6 pl-1",
        isTodoList ? "ml-0 list-none pl-0" : "list-outside"
      )}
      start={typeof element.listStart === "number" ? element.listStart : undefined}
      style={style}
    >
      <li>{props.children}</li>
    </ListTag>
  )
}
