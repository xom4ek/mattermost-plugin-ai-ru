import React from 'react';
import styled from 'styled-components';

const ThreadItemContainer = styled.div`
    padding: 16px;
    cursor: pointer;
    border-bottom: 1px solid rgba(var(--center-channel-color-rgb), 0.12)
`;

const Timestamp = (window as any).Components.Timestamp;

const Title = styled.div`
    color: var(--center-channel-color);
    display: flex;
    align-items: center;
    margin-bottom: 4px;
    justify-content: space-between;
`;

const TitleText = styled.div`
    font-size: 14px;
    font-weight: 600;
    text-overflow: ellipsis;
    overflow: hidden;
    white-space: nowrap;
`;

const Preview = styled.div`
    overflow: hidden;
    color: var(--center-channel-color);
    text-overflow: ellipsis;
    whitespace: nowrap;
    margin-bottom: 12px;
	height: 40px;
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
`;

const RepliesCount = styled.div`
    color: rgba(var(--center-channel-color-rgb), 0.64);
    font-weight: 600;
`;

const LastActivityDate = styled.div`
    color: rgba(var(--center-channel-color-rgb), 0.64);
    font-size: 12px;
    font-weight: 400;
    white-space: nowrap;
    margin-left: 13px;
`;

type Props = {
    postTitle: string;
    postMessage: string;
    repliesCount: number;
    lastActivityDate: number;
    onClick: () => void;
}

const DefaultTitle = 'Conversation with AI Assistant';

export default function ThreadItem(props: Props) {
    return (
        <ThreadItemContainer onClick={props.onClick}>
            <Title>
                <TitleText>{props.postTitle || DefaultTitle}</TitleText>
                <LastActivityDate>
                    <Timestamp // Matches the timestap format in the threads view
                        value={props.lastActivityDate}
                        units={['now', 'minute', 'hour', 'day', 'week']}
                        useTime={false}
                        day={'numeric'}
                    />
                </LastActivityDate>
            </Title>
            <Preview>{props.postMessage}</Preview>
            <RepliesCount>{props.repliesCount}{' replies'}</RepliesCount>
        </ThreadItemContainer>
    );
}
