import React, { SyntheticEvent, useCallback, useEffect, useState } from 'react';

import { CoreApp, LoadingState } from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import { EditorHeader, EditorRows, FlexItem, Space, Stack } from '@grafana/experimental';
import { reportInteraction } from '@grafana/runtime';
import { Button, ConfirmModal } from '@grafana/ui';
import { QueryEditorModeToggle } from 'app/plugins/datasource/prometheus/querybuilder/shared/QueryEditorModeToggle';
import { QueryHeaderSwitch } from 'app/plugins/datasource/prometheus/querybuilder/shared/QueryHeaderSwitch';
import { QueryEditorMode } from 'app/plugins/datasource/prometheus/querybuilder/shared/types';

import { lokiQueryEditorExplainKey, useFlag } from '../../prometheus/querybuilder/shared/hooks/useFlag';
import { LabelBrowserModal } from '../querybuilder/components/LabelBrowserModal';
import { LokiQueryBuilderContainer } from '../querybuilder/components/LokiQueryBuilderContainer';
import { LokiQueryBuilderOptions } from '../querybuilder/components/LokiQueryBuilderOptions';
import { LokiQueryCodeEditor } from '../querybuilder/components/LokiQueryCodeEditor';
import { QueryPatternsModal } from '../querybuilder/components/QueryPatternsModal';
import { buildVisualQueryFromString } from '../querybuilder/parsing';
import { changeEditorMode, getQueryWithDefaults } from '../querybuilder/state';
import { LokiQuery } from '../types';

import { LokiQueryEditorProps } from './types';

export const testIds = {
  editor: 'loki-editor',
};

export const LokiQueryEditor = React.memo<LokiQueryEditorProps>((props) => {
  const { onChange, onRunQuery, onAddQuery, data, app, queries, datasource } = props;
  const [parseModalOpen, setParseModalOpen] = useState(false);
  const [queryPatternsModalOpen, setQueryPatternsModalOpen] = useState(false);
  const [dataIsStale, setDataIsStale] = useState(false);
  const [labelBrowserVisible, setLabelBrowserVisible] = useState(false);
  const [labelsLoaded, setLabelsLoaded] = useState(false);
  const { flag: explain, setFlag: setExplain } = useFlag(lokiQueryEditorExplainKey);

  const query = getQueryWithDefaults(props.query);
  // This should be filled in from the defaults by now.
  const editorMode = query.editorMode!;

  const onExplainChange = (event: SyntheticEvent<HTMLInputElement>) => {
    setExplain(event.currentTarget.checked);
  };

  const onEditorModeChange = useCallback(
    (newEditorMode: QueryEditorMode) => {
      reportInteraction('grafana_loki_editor_mode_clicked', {
        newEditor: newEditorMode,
        previousEditor: query.editorMode ?? '',
        newQuery: !query.expr,
        app: app ?? '',
      });

      if (newEditorMode === QueryEditorMode.Builder) {
        const result = buildVisualQueryFromString(query.expr || '');
        // If there are errors, give user a chance to decide if they want to go to builder as that can lose some data.
        if (result.errors.length) {
          setParseModalOpen(true);
          return;
        }
      }
      changeEditorMode(query, newEditorMode, onChange);
    },
    [onChange, query, app]
  );

  useEffect(() => {
    setDataIsStale(false);
  }, [data]);

  const onChangeInternal = (query: LokiQuery) => {
    setDataIsStale(true);
    onChange(query);
  };

  const onClickChooserButton = () => {
    setLabelBrowserVisible((visible) => !visible);
  };

  const getChooserText = (logLabelsLoaded: boolean, hasLogLabels: boolean) => {
    if (!logLabelsLoaded) {
      return 'Loading labels...';
    }
    if (!hasLogLabels) {
      return '(No labels found)';
    }
    return 'Label browser';
  };

  useEffect(() => {
    datasource.languageProvider.start().then(() => {
      setLabelsLoaded(true);
    });
  }, [datasource]);

  const hasLogLabels = datasource.languageProvider.getLabelKeys().length > 0;
  const labelBrowserText = getChooserText(labelsLoaded, hasLogLabels);
  const buttonDisabled = !(labelsLoaded && hasLogLabels);

  return (
    <>
      <ConfirmModal
        isOpen={parseModalOpen}
        title="Query parsing"
        body="There were errors while trying to parse the query. Continuing to visual builder may lose some parts of the query."
        confirmText="Continue"
        onConfirm={() => {
          onChange({ ...query, editorMode: QueryEditorMode.Builder });
          setParseModalOpen(false);
        }}
        onDismiss={() => setParseModalOpen(false)}
      />
      <QueryPatternsModal
        isOpen={queryPatternsModalOpen}
        onClose={() => setQueryPatternsModalOpen(false)}
        query={query}
        queries={queries}
        app={app}
        onChange={onChange}
        onAddQuery={onAddQuery}
      />
      <EditorHeader>
        <LabelBrowserModal
          isOpen={labelBrowserVisible}
          languageProvider={datasource.languageProvider}
          query={query}
          app={app}
          onClose={() => setLabelBrowserVisible(false)}
          onChange={onChangeInternal}
          onRunQuery={onRunQuery}
        />
        <Stack gap={1}>
          <Button
            aria-label={selectors.components.QueryBuilder.queryPatterns}
            variant="secondary"
            size="sm"
            onClick={() => {
              setQueryPatternsModalOpen((prevValue) => !prevValue);

              const visualQuery = buildVisualQueryFromString(query.expr || '');
              reportInteraction('grafana_loki_query_patterns_opened', {
                version: 'v2',
                app: app ?? '',
                editorMode: query.editorMode,
                preSelectedOperationsCount: visualQuery.query.operations.length,
                preSelectedLabelsCount: visualQuery.query.labels.length,
              });
            }}
          >
            Kick start your query
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={onClickChooserButton}
            disabled={buttonDisabled}
            data-testid="label-browser-button"
          >
            {labelBrowserText}
          </Button>
        </Stack>
        <QueryHeaderSwitch label="Explain" value={explain} onChange={onExplainChange} />
        <FlexItem grow={1} />
        {app !== CoreApp.Explore && (
          <Button
            variant={dataIsStale ? 'primary' : 'secondary'}
            size="sm"
            onClick={onRunQuery}
            icon={data?.state === LoadingState.Loading ? 'fa fa-spinner' : undefined}
            disabled={data?.state === LoadingState.Loading}
          >
            Run queries
          </Button>
        )}
        <QueryEditorModeToggle mode={editorMode!} onChange={onEditorModeChange} />
      </EditorHeader>
      <Space v={0.5} />
      <EditorRows>
        {editorMode === QueryEditorMode.Code && (
          <LokiQueryCodeEditor {...props} query={query} onChange={onChangeInternal} showExplain={explain} />
        )}
        {editorMode === QueryEditorMode.Builder && (
          <LokiQueryBuilderContainer
            datasource={props.datasource}
            query={query}
            onChange={onChangeInternal}
            onRunQuery={props.onRunQuery}
            showExplain={explain}
          />
        )}
        <LokiQueryBuilderOptions query={query} onChange={onChange} onRunQuery={onRunQuery} app={app} />
      </EditorRows>
    </>
  );
});

LokiQueryEditor.displayName = 'LokiQueryEditor';
