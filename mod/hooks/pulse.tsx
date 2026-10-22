import type { ClientModule } from 'claude-code'

// colors runs from the text's own color to the color of the letter at the center of the band.
export type PulseProps = { text: string; colors: string[] }
type Local = { phase: number }

const TICK_MS = 60
// Frames with no band between two passes, about half a second
const PAUSE_TICKS = 8

// Pulse draws text with a shimmer: a narrow band of lighter letters that moves across
// the text from the first letter to the last, one letter per frame, then pauses.
const Pulse: ClientModule<PulseProps, Local> = (props, surface) => {
  const { Text } = surface.elements
  if (surface.state === undefined) {
    surface.setState({ phase: 0 })
    surface.every(TICK_MS, () => surface.setState({ phase: (surface.state?.phase ?? 0) + 1 }))
  }
  const chars = [...props.text]
  const top = props.colors.length - 1
  // The band enters before the first letter and leaves after the last
  const band = ((surface.state?.phase ?? 0) % (chars.length + 2 * top + PAUSE_TICKS)) - top
  return (
    <Text wrap="truncate-end">
      {chars.map((ch, i) => (
        <Text bold color={props.colors[Math.max(0, top - Math.abs(i - band))]}>
          {ch}
        </Text>
      ))}
    </Text>
  )
}

export default Pulse
