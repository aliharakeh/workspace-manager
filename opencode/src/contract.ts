import { Schema } from 'effect';
import { Ipc } from '@opencode/plugin/gui';

export const WorkspaceManager = Ipc.define({
  id: 'workspace-manager-opencode.daemon',
  state: Schema.Struct({ connected: Schema.Boolean }),
  methods: {
    connect: {},
    snapshot: { input: Schema.String, output: Schema.Unknown },
    act: {
      input: Schema.Struct({ dir: Schema.String, action: Schema.String, args: Schema.Array(Schema.Unknown), expectedSetId: Schema.optional(Schema.Number) }),
      output: Schema.Unknown,
    },
    add: { input: Schema.Struct({ dir: Schema.String, workspace: Schema.String }), output: Schema.Unknown },
    preview: {
      input: Schema.Struct({ configSetId: Schema.Number, content: Schema.String }),
      output: Schema.String,
    },
    output: {
      input: Schema.Struct({ appID: Schema.Number, commandID: Schema.Number }),
      output: Schema.Struct({ data: Schema.String, end: Schema.Number }),
    },
    resize: {
      input: Schema.Struct({ appID: Schema.Number, commandID: Schema.Number, cols: Schema.Number, rows: Schema.Number }),
    },
  },
  events: { changed: Schema.Void, runner: Schema.Unknown },
});
