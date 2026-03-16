import { toTitleCase } from "@/lib/utils"

const STATEFUL_IMAGE_KEYWORDS = [
  "mysql",
  "mariadb",
  "postgres",
  "postgresql",
  "pgsql",
  "redis",
  "mongodb",
  "mongo",
  "clickhouse",
  "elasticsearch",
  "opensearch",
  "kafka",
  "zookeeper",
  "rabbitmq",
  "etcd",
  "consul",
  "minio",
  "influxdb",
] as const

const normalizeImageName = (imageName: string) => {
  return imageName.trim().toLowerCase()
}

export const extractImageName = (image: string) => {
  const withoutDigest = image.trim().split("@")[0] ?? ""
  const lastSegment = withoutDigest.split("/").pop() ?? ""
  return lastSegment.split(":")[0] ?? ""
}

export const toImageSlug = (imageName: string) => {
  return normalizeImageName(imageName)
    .replace(/[._\s]+/g, "-")
    .replace(/[^a-z0-9-]/g, "")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")
}

export const toNameSlug = (name: string) => {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")
}

export const toImageTitle = (imageName: string) => {
  const readable = imageName
    .trim()
    .replace(/[._-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim()

  return readable ? toTitleCase(readable) : ""
}

export const isStatefulImage = (image: string) => {
  const normalizedImage = image.trim().toLowerCase()
  return STATEFUL_IMAGE_KEYWORDS.some((keyword) => normalizedImage.includes(keyword))
}

export const deriveImageDefaults = (image: string) => {
  const imageName = extractImageName(image)

  return {
    imageName,
    slug: toImageSlug(imageName),
    name: toImageTitle(imageName),
    isStateful: isStatefulImage(image),
  }
}
