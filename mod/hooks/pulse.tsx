import type { ClientModule } from 'claude-code'

// colors runs from the text's own color to the color at the center of the band.
export type PulseProps = { text: string; colors: string[] }
type Local = { phase: number }

const TICK_MS = 90

// Pulse draws text with a bright band that moves across it, one character per frame.
const Pulse: ClientModule<PulseProps, Local> = (props, surface) => {
  const { Text } = surface.elements
  if (surface.state === undefined) {
    surface.setState({ phase: 0 })
    surface.every(TICK_MS, () => surface.setState({ phase: (surface.state?.phase ?? 0) + 1 }))
  }
  const chars = [...props.text]
  const top = props.colors.length - 1
  // The band starts before the first character and ends after the last
  const band = ((surface.state?.phase ?? 0) % (chars.length + 2 * top)) - top
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
