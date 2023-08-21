import {Store, Action} from 'redux';

import {GlobalState} from '@mattermost/types/lib/store';

import {manifest} from '@/manifest';

import {LLMBotPost} from './components/llmbot_post';
import {doReaction, doSummarize, doJiraTicket, doTranscribe} from './client';

const BotUsername = 'ai';

export default class Plugin {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars, @typescript-eslint/no-empty-function
    public async initialize(registry: any, store: Store<GlobalState, Action<Record<string, unknown>>>) {
        registry.registerPostDropdownMenuAction('React for me', doReaction);
        registry.registerPostDropdownMenuAction('Summarize Thread', makeSummarizePost(store));
        registry.registerPostDropdownMenuAction('Создай Jira ticket', makeJiraTicket(store));
        registry.registerPostTypeComponent('custom_llmbot', LLMBotPost);
        registry.registerPostDropdownMenuAction('Summarize Meeting Audio', doTranscribe);
    }
}

function makeSummarizePost(store: Store<GlobalState, Action<Record<string, unknown>>>) {
    return async function summarizePost(postid: string) {
        const state = store.getState();
        const team = state.entities.teams.teams[state.entities.teams.currentTeamId];
        window.WebappUtils.browserHistory.push('/' + team.name + '/messages/@' + BotUsername);

        await doSummarize(postid);
    };
}

function makeJiraTicket(store: Store<GlobalState, Action<Record<string, unknown>>>) {
    return async function JiraTicket(postid: string) {
        const state = store.getState();
        const team = state.entities.teams.teams[state.entities.teams.currentTeamId];
        window.WebappUtils.browserHistory.push('/' + team.name + '/messages/@' + BotUsername);

        await doJiraTicket(postid);
    };
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void
        WebappUtils: any
    }
}

window.registerPlugin(manifest.id, new Plugin());
