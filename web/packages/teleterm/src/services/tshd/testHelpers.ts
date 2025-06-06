/**
 * Teleport
 * Copyright (C) 2023  Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

import { TrustedDeviceRequirement } from 'gen-proto-ts/teleport/legacy/types/trusted_device_requirement_pb';
import { App } from 'gen-proto-ts/teleport/lib/teleterm/v1/app_pb';
import {
  AuthSettings,
  ClientVersionStatus,
} from 'gen-proto-ts/teleport/lib/teleterm/v1/auth_settings_pb';
import {
  ACL,
  ShowResources,
} from 'gen-proto-ts/teleport/lib/teleterm/v1/cluster_pb';
import { WindowsDesktop } from 'gen-proto-ts/teleport/lib/teleterm/v1/windows_desktop_pb';

import { TshdRpcError } from './cloneableClient';
import * as tsh from './types';

export const rootClusterUri = '/clusters/teleport-local.com';
export const leafClusterUri = `${rootClusterUri}/leaves/leaf`;

export const makeServer = (props: Partial<tsh.Server> = {}): tsh.Server => ({
  uri: `${rootClusterUri}/servers/1234abcd-1234-abcd-1234-abcd1234abcd`,
  tunnel: false,
  name: '1234abcd-1234-abcd-1234-abcd1234abcd',
  hostname: 'foo',
  addr: '127.0.0.1:3022',
  labels: [],
  subKind: 'teleport',
  ...props,
});

export const databaseUri = `${rootClusterUri}/dbs/foo`;
export const kubeUri = `${rootClusterUri}/kubes/foo`;
export const appUri = `${rootClusterUri}/apps/foo`;
export const windowsDesktopUri = `${rootClusterUri}/windows_desktops/foo`;

export const makeDatabase = (
  props: Partial<tsh.Database> = {}
): tsh.Database => ({
  uri: databaseUri,
  name: 'foo',
  protocol: 'postgres',
  type: 'self-hosted',
  desc: '',
  hostname: '',
  addr: '',
  labels: [],
  ...props,
});

export const makeKube = (props: Partial<tsh.Kube> = {}): tsh.Kube => ({
  name: 'foo',
  labels: [],
  uri: `${rootClusterUri}/kubes/foo`,
  ...props,
});

export const makeApp = (props: Partial<App> = {}): App => ({
  name: 'foo',
  labels: [],
  endpointUri: 'tcp://localhost:3000',
  friendlyName: '',
  desc: '',
  awsConsole: false,
  publicAddr: 'local-app.example.com',
  fqdn: 'local-app.example.com',
  samlApp: false,
  uri: appUri,
  awsRoles: [],
  tcpPorts: [],
  ...props,
});

export const makeWindowsDesktop = (
  props: Partial<WindowsDesktop> = {}
): WindowsDesktop => ({
  uri: windowsDesktopUri,
  name: 'windows-server-2019',
  labels: [],
  addr: '192.169.100.50',
  logins: ['Administrator'],
  ...props,
});

export const makeLabelsList = (labels: Record<string, string>): tsh.Label[] =>
  Object.entries(labels).map(([name, value]) => ({ name, value }));

export const makeRootCluster = (
  props: Partial<tsh.Cluster> = {}
): tsh.Cluster => ({
  uri: rootClusterUri,
  name: 'teleport-local',
  connected: true,
  leaf: false,
  proxyHost: 'teleport-local.com:3080',
  authClusterId: 'fefe3434-fefe-3434-fefe-3434fefe3434',
  loggedInUser: makeLoggedInUser(),
  proxyVersion: '11.1.0',
  showResources: ShowResources.REQUESTABLE,
  profileStatusError: '',
  ssoHost: 'example.auth0.com',
  ...props,
});

export const makeLeafCluster = (
  props: Partial<tsh.Cluster> = {}
): tsh.Cluster => ({
  uri: leafClusterUri,
  name: 'teleport-local-leaf',
  connected: true,
  leaf: true,
  proxyHost: '',
  authClusterId: '',
  loggedInUser: makeLoggedInUser(),
  proxyVersion: '',
  profileStatusError: '',
  showResources: ShowResources.UNSPECIFIED,
  ssoHost: 'example.auth0.com',
  ...props,
});

export const makeAcl = (props: Partial<ACL> = {}): ACL => ({
  recordedSessions: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  activeSessions: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  authConnectors: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  roles: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  users: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  trustedClusters: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  events: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  tokens: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  servers: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  apps: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  dbs: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  kubeservers: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  accessRequests: {
    list: true,
    read: true,
    edit: true,
    create: true,
    delete: true,
    use: true,
  },
  reviewRequests: true,
  directorySharingEnabled: true,
  clipboardSharingEnabled: true,
  ...props,
});

export const makeLoggedInUser = (
  props: Partial<tsh.LoggedInUser> = {}
): tsh.LoggedInUser => ({
  activeRequests: [],
  name: 'alice',
  isDeviceTrusted: false,
  trustedDeviceRequirement: TrustedDeviceRequirement.NOT_REQUIRED,
  acl: makeAcl(),
  sshLogins: [],
  roles: [],
  requestableRoles: [],
  suggestedReviewers: [],
  userType: tsh.LoggedInUser_UserType.LOCAL,
  ...props,
});

export const makeDatabaseGateway = (
  props: Partial<tsh.Gateway> = {}
): tsh.Gateway => ({
  uri: '/gateways/db',
  targetName: 'sales-production',
  targetUri: databaseUri,
  targetUser: 'alice',
  localAddress: 'localhost',
  localPort: '1337',
  protocol: 'postgres',
  gatewayCliCommand: {
    path: '/foo/psql',
    args: ['psql', 'localhost:1337'],
    env: [],
    preview: 'psql localhost:1337',
  },
  targetSubresourceName: 'bar',
  ...props,
});

export const makeKubeGateway = (
  props: Partial<tsh.Gateway> = {}
): tsh.Gateway => ({
  uri: '/gateways/kube',
  targetName: 'foo',
  targetUri: kubeUri,
  targetUser: '',
  localAddress: 'localhost',
  localPort: '1337',
  protocol: '',
  gatewayCliCommand: {
    path: '/bin/kubectl',
    args: ['version'],
    env: ['KUBECONFIG=/path/to/kubeconfig'],
    preview: 'KUBECONFIG=/path/to/kubeconfig /bin/kubectl version',
  },
  targetSubresourceName: '',
  ...props,
});

export const makeAppGateway = (
  props: Partial<tsh.Gateway> = {}
): tsh.Gateway => ({
  uri: '/gateways/app',
  targetName: 'sales-production',
  targetUri: appUri,
  localAddress: 'localhost',
  localPort: '1337',
  targetSubresourceName: undefined,
  gatewayCliCommand: {
    path: '',
    preview: 'curl http://localhost:1337',
    env: [],
    args: [],
  },
  targetUser: '',
  protocol: 'HTTP',
  ...props,
});

export const makeRetryableError = (): TshdRpcError => ({
  name: 'TshdRpcError',
  isResolvableWithRelogin: true,
  code: 'UNKNOWN',
  message: 'ssh: handshake failed',
});

export const makeAccessRequest = (
  props: Partial<tsh.AccessRequest> = {}
): tsh.AccessRequest => ({
  id: '01929070-6886-77eb-90aa-c7223dd73f67',
  state: 'APPROVED',
  resolveReason: '',
  requestReason: '',
  user: makeLoggedInUser().name,
  roles: ['access', 'searcheable-resources'],
  reviews: [],
  suggestedReviewers: ['admin', 'reviewer'],
  thresholdNames: ['default'],
  resourceIds: [
    {
      kind: 'kube_cluster',
      name: 'minikube',
      clusterName: 'main',
      subResourceName: '',
    },
  ],
  resources: [
    {
      id: {
        kind: 'kube_cluster',
        name: 'minikube',
        clusterName: 'main',
        subResourceName: '',
      },
      details: { hostname: '', friendlyName: '' },
    },
  ],
  promotedAccessListTitle: '',
  created: { seconds: 1729000138n, nanos: 886521000 },
  expires: { seconds: 1729026573n, nanos: 0 },
  maxDuration: { seconds: 1729026573n, nanos: 0 },
  requestTtl: { seconds: 1729026573n, nanos: 0 },
  sessionTtl: { seconds: 1729026573n, nanos: 0 },
  ...props,
});

export const makeAuthSettings = (
  props: Partial<AuthSettings> = {}
): AuthSettings => ({
  localAuthEnabled: true,
  authProviders: [],
  hasMessageOfTheDay: false,
  authType: 'local',
  allowPasswordless: false,
  localConnectorName: '',
  clientVersionStatus: ClientVersionStatus.OK,
  ...props,
});
