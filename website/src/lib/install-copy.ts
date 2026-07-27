export type InstallCopyTracker = (
  eventName: 'install_command_copy',
  target: 'homepage-hero',
) => boolean | void;

export async function copyInstallCommand(
  command: string,
  writeText: (value: string) => Promise<void>,
  track?: InstallCopyTracker,
): Promise<void> {
  await writeText(command);
  track?.('install_command_copy', 'homepage-hero');
}
