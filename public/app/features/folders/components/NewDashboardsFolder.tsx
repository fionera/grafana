import React from 'react';
import { connect, ConnectedProps } from 'react-redux';

import { NavModelItem } from '@grafana/data';
import { config } from '@grafana/runtime';
import { Button, Input, Form, Field } from '@grafana/ui';
import { Page } from 'app/core/components/Page/Page';
import { useQueryParams } from 'app/core/hooks/useQueryParams';

import { validationSrv } from '../../manage-dashboards/services/ValidationSrv';
import { createNewFolder } from '../state/actions';

const mapDispatchToProps = {
  createNewFolder,
};

const connector = connect(null, mapDispatchToProps);

interface OwnProps {}

interface FormModel {
  folderName: string;
}

type Props = OwnProps & ConnectedProps<typeof connector>;

const initialFormModel: FormModel = { folderName: '' };

const pageNav: NavModelItem = {
  text: 'Create a new folder',
  subTitle: 'Folders provide a way to group dashboards and alert rules.',
  breadcrumbs: [{ title: 'Dashboards', url: 'dashboards' }],
};

function NewDashboardsFolder({ createNewFolder }: Props) {
  const [queryParams] = useQueryParams();
  const onSubmit = (formData: FormModel) => {
    createNewFolder(formData.folderName, String(queryParams['folderUid']));
  };
  const validateFolderName = (folderName: string) => {
    return validationSrv
      .validateNewFolderName(folderName)
      .then(() => {
        return true;
      })
      .catch((e) => {
        return e.message;
      });
  };

  return (
    <Page navId="dashboards/browse" pageNav={pageNav}>
      <Page.Contents>
        {!config.featureToggles.topnav && <h3>New dashboard folder</h3>}
        <Form defaultValues={initialFormModel} onSubmit={onSubmit}>
          {({ register, errors }) => (
            <>
              <Field
                label="Folder name"
                invalid={!!errors.folderName}
                error={errors.folderName && errors.folderName.message}
              >
                <Input
                  id="folder-name-input"
                  {...register('folderName', {
                    required: 'Folder name is required.',
                    validate: async (v) => await validateFolderName(v),
                  })}
                />
              </Field>
              <Button type="submit">Create</Button>
            </>
          )}
        </Form>
      </Page.Contents>
    </Page>
  );
}

export default connector(NewDashboardsFolder);
