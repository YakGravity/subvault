# Migration from SubTrackr

If you're migrating from [SubTrackr](https://github.com/bscott/subtrackr):

## Steps

1. **Rename the database** — Rename `subtrackr.db` to `subvault.db`, or set `DATABASE_PATH` to point to your existing file
2. **Update Docker volumes** — Change volume names in your Docker Compose file from `subtrackr_data` to `subvault_data`
3. **Theme reset** — Theme preferences stored in localStorage will reset (new key names)
4. **Session reset** — Users will need to log in again (new session keys)

## Import Compatibility

SubVault can import both SubTrackr and SubVault backup formats. Use **Settings > Import/Export** to restore a SubTrackr backup directly into SubVault.
